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
	"github.com/kareemaly/cortex/internal/storage"
)

// Store manages doc storage with directory-per-entity organized by category.
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
		DocMeta: DocMeta{
			ID:         uuid.New().String(),
			Title:      title,
			Tags:       tags,
			References: references,
			Created:    now,
			Updated:    now,
		},
		Category: category,
		Body:     body,
		Comments: []Comment{},
	}

	// Ensure category subdirectory exists
	catDir := filepath.Join(s.docsDir, category)
	if err := os.MkdirAll(catDir, 0755); err != nil {
		return nil, fmt.Errorf("create category directory: %w", err)
	}

	mu := s.docMu(doc.ID)
	mu.Lock()
	defer mu.Unlock()

	if err := s.saveDoc(doc); err != nil {
		return nil, fmt.Errorf("save doc: %w", err)
	}

	s.emit(events.DocCreated, doc.ID)
	return doc, nil
}

// Get retrieves a doc by ID, scanning all category subdirectories.
// Loads comments from comment-*.md files.
func (s *Store) Get(id string) (*Doc, error) {
	entityDir, err := s.findEntityDir(id)
	if err != nil {
		return nil, err
	}

	doc, err := s.loadIndex(entityDir)
	if err != nil {
		return nil, err
	}

	comments, err := storage.ListComments(entityDir)
	if err != nil {
		return nil, fmt.Errorf("load comments: %w", err)
	}
	doc.Comments = comments

	return doc, nil
}

// Update modifies a doc's fields. Only non-nil parameters are applied.
func (s *Store) Update(id string, title, body *string, tags, references *[]string) (*Doc, error) {
	mu := s.docMu(id)
	mu.Lock()
	defer mu.Unlock()

	entityDir, err := s.findEntityDir(id)
	if err != nil {
		return nil, err
	}

	doc, err := s.loadIndex(entityDir)
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

	if titleChanged {
		// Title change means slug changes → rename directory
		newDirName := storage.DirName(doc.Title, doc.ID, "doc")
		newDir := filepath.Join(s.docsDir, doc.Category, newDirName)
		if err := os.Rename(entityDir, newDir); err != nil {
			return nil, fmt.Errorf("rename entity dir: %w", err)
		}
		entityDir = newDir
	}

	if err := s.writeIndex(entityDir, doc); err != nil {
		return nil, fmt.Errorf("save doc: %w", err)
	}

	s.emit(events.DocUpdated, doc.ID)
	return doc, nil
}

// Delete removes a doc by ID (entire entity directory).
func (s *Store) Delete(id string) error {
	mu := s.docMu(id)
	mu.Lock()
	defer mu.Unlock()

	entityDir, err := s.findEntityDir(id)
	if err != nil {
		return err
	}

	if err := os.RemoveAll(entityDir); err != nil {
		return fmt.Errorf("remove entity directory: %w", err)
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

	entityDir, err := s.findEntityDir(id)
	if err != nil {
		return nil, err
	}

	doc, err := s.loadIndex(entityDir)
	if err != nil {
		return nil, err
	}

	if doc.Category == category {
		return doc, nil
	}

	// Ensure new category subdirectory exists
	newCatDir := filepath.Join(s.docsDir, category)
	if err := os.MkdirAll(newCatDir, 0755); err != nil {
		return nil, fmt.Errorf("create category directory: %w", err)
	}

	// Move entity directory to new category
	dirName := filepath.Base(entityDir)
	newDir := filepath.Join(newCatDir, dirName)
	if err := os.Rename(entityDir, newDir); err != nil {
		return nil, fmt.Errorf("move entity dir: %w", err)
	}

	doc.Category = category
	doc.Updated = time.Now().UTC()

	if err := s.writeIndex(newDir, doc); err != nil {
		return nil, fmt.Errorf("save doc: %w", err)
	}

	s.emit(events.DocUpdated, doc.ID)
	return doc, nil
}

// List returns docs with optional filtering by category, tag, and query.
// Does NOT load comments for performance.
func (s *Store) List(category, tag, query string) ([]*Doc, error) {
	query = strings.ToLower(query)

	var catDirs []string
	if category != "" {
		catDirs = []string{filepath.Join(s.docsDir, category)}
	} else {
		entries, err := os.ReadDir(s.docsDir)
		if err != nil {
			if os.IsNotExist(err) {
				return []*Doc{}, nil
			}
			return nil, fmt.Errorf("read docs directory: %w", err)
		}
		for _, entry := range entries {
			if entry.IsDir() && !strings.HasPrefix(entry.Name(), ".") {
				catDirs = append(catDirs, filepath.Join(s.docsDir, entry.Name()))
			}
		}
	}

	var docs []*Doc
	for _, catDir := range catDirs {
		entries, err := os.ReadDir(catDir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("read directory %s: %w", catDir, err)
		}

		for _, entry := range entries {
			if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
				continue
			}

			entityDir := filepath.Join(catDir, entry.Name())
			doc, err := s.loadIndex(entityDir)
			if err != nil {
				return nil, err
			}

			if tag != "" && !containsTag(doc.Tags, tag) {
				continue
			}

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

// AddComment adds a comment to a doc.
func (s *Store) AddComment(docID, author string, commentType CommentType, content string, action *storage.CommentAction) (*Comment, error) {
	mu := s.docMu(docID)
	mu.Lock()
	defer mu.Unlock()

	entityDir, err := s.findEntityDir(docID)
	if err != nil {
		return nil, err
	}

	comment, err := storage.CreateComment(entityDir, author, commentType, content, action)
	if err != nil {
		return nil, err
	}

	// Update the doc's updated timestamp
	doc, err := s.loadIndex(entityDir)
	if err != nil {
		return nil, err
	}
	doc.Updated = time.Now().UTC()
	if err := s.writeIndex(entityDir, doc); err != nil {
		return nil, fmt.Errorf("save doc: %w", err)
	}

	s.emit(events.DocUpdated, docID)
	return comment, nil
}

// ListComments returns all comments for a doc sorted by created time.
func (s *Store) ListComments(docID string) ([]Comment, error) {
	entityDir, err := s.findEntityDir(docID)
	if err != nil {
		return nil, err
	}
	return storage.ListComments(entityDir)
}

// saveDoc creates the entity directory and writes index.md.
func (s *Store) saveDoc(doc *Doc) error {
	dirName := storage.DirName(doc.Title, doc.ID, "doc")
	entityDir := filepath.Join(s.docsDir, doc.Category, dirName)

	if err := os.MkdirAll(entityDir, 0755); err != nil {
		return fmt.Errorf("create entity dir: %w", err)
	}

	return s.writeIndex(entityDir, doc)
}

// writeIndex writes the index.md file in the given entity directory.
func (s *Store) writeIndex(entityDir string, doc *Doc) error {
	data, err := storage.SerializeFrontmatter(&doc.DocMeta, doc.Body)
	if err != nil {
		return fmt.Errorf("serialize doc: %w", err)
	}

	target := filepath.Join(entityDir, "index.md")
	return storage.AtomicWriteFile(target, data)
}

// loadIndex reads and parses index.md from the given entity directory.
// Derives Category from the parent directory name.
func (s *Store) loadIndex(entityDir string) (*Doc, error) {
	data, err := os.ReadFile(filepath.Join(entityDir, "index.md"))
	if err != nil {
		return nil, fmt.Errorf("read index.md: %w", err)
	}

	meta, body, err := storage.ParseFrontmatter[DocMeta](data)
	if err != nil {
		return nil, fmt.Errorf("parse index.md: %w", err)
	}

	// Category is derived from the parent directory of the entity dir
	category := filepath.Base(filepath.Dir(entityDir))

	return &Doc{
		DocMeta:  *meta,
		Category: category,
		Body:     body,
		Comments: []Comment{},
	}, nil
}

// findEntityDir finds the entity directory for a doc by scanning all category subdirectories.
func (s *Store) findEntityDir(id string) (string, error) {
	shortID := storage.ShortID(id)

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
			if !catEntry.IsDir() {
				continue
			}
			name := catEntry.Name()
			if strings.HasSuffix(name, "-"+shortID) || strings.HasSuffix(name, "-"+id) {
				return filepath.Join(catDir, name), nil
			}
		}
	}

	return "", &NotFoundError{Resource: "doc", ID: id}
}

// GetFilePath returns the filesystem path to a doc's index.md file.
func (s *Store) GetFilePath(id string) (string, error) {
	entityDir, err := s.findEntityDir(id)
	if err != nil {
		return "", err
	}
	return filepath.Join(entityDir, "index.md"), nil
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
