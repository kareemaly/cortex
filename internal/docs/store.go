package docs

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/kareemaly/cortex/internal/events"
)

// Store manages doc storage with markdown files organized by category subdirectories.
type Store struct {
	docsDir     string
	locks       sync.Map // maps doc ID → *sync.Mutex
	bus         *events.Bus
	projectPath string
}

// docMu returns the mutex for a given doc ID, creating one if needed.
func (s *Store) docMu(id string) *sync.Mutex {
	v, _ := s.locks.LoadOrStore(id, &sync.Mutex{})
	return v.(*sync.Mutex)
}

func (s *Store) emit(eventType events.EventType, docID string) {
	if s.bus == nil {
		return
	}
	s.bus.Emit(events.Event{
		Type:        eventType,
		ProjectPath: s.projectPath,
		TicketID:    docID, // Reuse TicketID field for doc IDs
	})
}

// NewStore creates a new Store and ensures the base directory exists.
// bus and projectPath are optional; pass nil/"" to disable event emission.
func NewStore(docsDir string, bus *events.Bus, projectPath string) (*Store, error) {
	if err := os.MkdirAll(docsDir, 0755); err != nil {
		return nil, fmt.Errorf("create docs directory %s: %w", docsDir, err)
	}
	return &Store{docsDir: docsDir, bus: bus, projectPath: projectPath}, nil
}

// Create creates a new doc in the given category.
func (s *Store) Create(title, category, body string, tags, references []string) (*Doc, error) {
	if title == "" {
		return nil, &ValidationError{Field: "title", Message: "cannot be empty"}
	}
	if category == "" {
		return nil, &ValidationError{Field: "category", Message: "cannot be empty"}
	}

	now := time.Now().UTC()
	doc := &Doc{
		ID:         uuid.New().String(),
		Title:      title,
		Category:   category,
		Tags:       tags,
		References: references,
		Created:    now,
		Updated:    now,
		Body:       body,
	}

	// Ensure category subdirectory exists
	catDir := filepath.Join(s.docsDir, category)
	if err := os.MkdirAll(catDir, 0755); err != nil {
		return nil, fmt.Errorf("create category directory: %w", err)
	}

	mu := s.docMu(doc.ID)
	mu.Lock()
	defer mu.Unlock()

	if err := s.save(doc); err != nil {
		return nil, fmt.Errorf("save doc: %w", err)
	}

	s.emit(events.DocCreated, doc.ID)
	return doc, nil
}

// Get retrieves a doc by ID, scanning all category subdirectories.
func (s *Store) Get(id string) (*Doc, error) {
	path, err := s.findDocPath(id)
	if err != nil {
		return nil, err
	}
	return s.loadFile(path)
}

// Update modifies a doc's fields. Only non-nil parameters are applied.
func (s *Store) Update(id string, title, body *string, tags, references *[]string) (*Doc, error) {
	mu := s.docMu(id)
	mu.Lock()
	defer mu.Unlock()

	path, err := s.findDocPath(id)
	if err != nil {
		return nil, err
	}

	doc, err := s.loadFile(path)
	if err != nil {
		return nil, err
	}

	titleChanged := false
	if title != nil {
		if *title == "" {
			return nil, &ValidationError{Field: "title", Message: "cannot be empty"}
		}
		if doc.Title != *title {
			titleChanged = true
		}
		doc.Title = *title
	}
	if body != nil {
		doc.Body = *body
	}
	if tags != nil {
		doc.Tags = *tags
	}
	if references != nil {
		doc.References = *references
	}

	doc.Updated = time.Now().UTC()

	// If title changed, slug changes → remove old file and save new
	if titleChanged {
		if err := os.Remove(path); err != nil {
			return nil, fmt.Errorf("remove old doc file: %w", err)
		}
	}

	if err := s.save(doc); err != nil {
		return nil, fmt.Errorf("save doc: %w", err)
	}

	s.emit(events.DocUpdated, doc.ID)
	return doc, nil
}

// Delete removes a doc by ID.
func (s *Store) Delete(id string) error {
	mu := s.docMu(id)
	mu.Lock()
	defer mu.Unlock()

	path, err := s.findDocPath(id)
	if err != nil {
		return err
	}

	if err := os.Remove(path); err != nil {
		return fmt.Errorf("remove doc file: %w", err)
	}

	s.locks.Delete(id)
	s.emit(events.DocDeleted, id)
	return nil
}

// Move moves a doc to a different category.
func (s *Store) Move(id, category string) (*Doc, error) {
	if category == "" {
		return nil, &ValidationError{Field: "category", Message: "cannot be empty"}
	}

	mu := s.docMu(id)
	mu.Lock()
	defer mu.Unlock()

	path, err := s.findDocPath(id)
	if err != nil {
		return nil, err
	}

	doc, err := s.loadFile(path)
	if err != nil {
		return nil, err
	}

	if doc.Category == category {
		return doc, nil // Already in target category
	}

	// Remove from old location
	if err := os.Remove(path); err != nil {
		return nil, fmt.Errorf("remove old doc file: %w", err)
	}

	// Update category and save to new location
	doc.Category = category
	doc.Updated = time.Now().UTC()

	// Ensure new category subdirectory exists
	catDir := filepath.Join(s.docsDir, category)
	if err := os.MkdirAll(catDir, 0755); err != nil {
		return nil, fmt.Errorf("create category directory: %w", err)
	}

	if err := s.save(doc); err != nil {
		return nil, fmt.Errorf("save doc: %w", err)
	}

	s.emit(events.DocUpdated, doc.ID)
	return doc, nil
}

// List returns docs with optional filtering by category, tag, and query.
func (s *Store) List(category, tag, query string) ([]*Doc, error) {
	query = strings.ToLower(query)

	var dirs []string
	if category != "" {
		// Only scan the specified category
		dirs = []string{filepath.Join(s.docsDir, category)}
	} else {
		// Scan all subdirectories
		entries, err := os.ReadDir(s.docsDir)
		if err != nil {
			if os.IsNotExist(err) {
				return []*Doc{}, nil
			}
			return nil, fmt.Errorf("read docs directory: %w", err)
		}
		for _, entry := range entries {
			if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
				dirs = append(dirs, filepath.Join(s.docsDir, entry.Name()))
			}
		}
	}

	var docs []*Doc
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("read directory %s: %w", dir, err)
		}

		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".md") || strings.HasPrefix(entry.Name(), ".tmp-") {
				continue
			}

			doc, err := s.loadFile(filepath.Join(dir, entry.Name()))
			if err != nil {
				return nil, err
			}

			// Apply tag filter
			if tag != "" && !containsTag(doc.Tags, tag) {
				continue
			}

			// Apply query filter (case-insensitive substring in title + body)
			if query != "" &&
				!strings.Contains(strings.ToLower(doc.Title), query) &&
				!strings.Contains(strings.ToLower(doc.Body), query) {
				continue
			}

			docs = append(docs, doc)
		}
	}

	if docs == nil {
		docs = []*Doc{}
	}
	return docs, nil
}

// filename generates the filename for a doc: {slug}-{shortID}.md
func (s *Store) filename(doc *Doc) string {
	slug := GenerateSlug(doc.Title)
	shortID := doc.ID
	if len(shortID) > 8 {
		shortID = shortID[:8]
	}
	return fmt.Sprintf("%s-%s.md", slug, shortID)
}

// save writes a doc to its category directory using atomic write.
func (s *Store) save(doc *Doc) error {
	dir := filepath.Join(s.docsDir, doc.Category)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create category directory: %w", err)
	}

	target := filepath.Join(dir, s.filename(doc))

	data, err := SerializeDoc(doc)
	if err != nil {
		return fmt.Errorf("serialize doc: %w", err)
	}

	// Write to a temp file in the same directory (same filesystem for atomic rename)
	tmp, err := os.CreateTemp(dir, ".tmp-*.md")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmp.Name()

	// Clean up temp file on error
	defer func() {
		if tmpPath != "" {
			_ = os.Remove(tmpPath)
		}
	}()

	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return fmt.Errorf("write temp file: %w", err)
	}

	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}

	if err := os.Rename(tmpPath, target); err != nil {
		return fmt.Errorf("rename temp file: %w", err)
	}

	// Rename succeeded — prevent deferred cleanup from removing the target
	tmpPath = ""

	return nil
}

// loadFile reads and parses a doc from a file.
func (s *Store) loadFile(path string) (*Doc, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	doc, err := ParseDoc(data)
	if err != nil {
		return nil, fmt.Errorf("parse doc %s: %w", path, err)
	}

	return doc, nil
}

// findDocPath finds the file path for a doc by scanning all category subdirectories.
func (s *Store) findDocPath(id string) (string, error) {
	shortID := id
	if len(shortID) > 8 {
		shortID = shortID[:8]
	}

	entries, err := os.ReadDir(s.docsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return "", &NotFoundError{Resource: "doc", ID: id}
		}
		return "", fmt.Errorf("read docs directory: %w", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		catDir := filepath.Join(s.docsDir, entry.Name())
		catEntries, err := os.ReadDir(catDir)
		if err != nil {
			continue
		}

		for _, catEntry := range catEntries {
			if catEntry.IsDir() {
				continue
			}
			name := catEntry.Name()
			if strings.HasSuffix(name, "-"+shortID+".md") || strings.HasSuffix(name, "-"+id+".md") {
				return filepath.Join(catDir, name), nil
			}
		}
	}

	return "", &NotFoundError{Resource: "doc", ID: id}
}

// containsTag checks if a slice of tags contains a specific tag (case-insensitive).
func containsTag(tags []string, tag string) bool {
	tag = strings.ToLower(tag)
	for _, t := range tags {
		if strings.ToLower(t) == tag {
			return true
		}
	}
	return false
}
