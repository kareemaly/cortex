package conclusion

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

// Store manages conclusion storage.
type Store struct {
	sessionsDir string
	mu          sync.RWMutex
	bus         *events.Bus
	projectPath string
}

// NewStore creates a new Store and ensures the directory exists.
func NewStore(sessionsDir string, bus *events.Bus, projectPath string) (*Store, error) {
	s := &Store{sessionsDir: sessionsDir, bus: bus, projectPath: projectPath}

	if err := os.MkdirAll(sessionsDir, 0755); err != nil {
		return nil, fmt.Errorf("create sessions directory %s: %w", sessionsDir, err)
	}

	return s, nil
}

func (s *Store) emit(eventType events.EventType, payload any) {
	if s.bus == nil {
		return
	}
	s.bus.Emit(events.Event{
		Type:        eventType,
		ProjectPath: s.projectPath,
		Payload:     payload,
	})
}

// Create creates a new conclusion record.
func (s *Store) Create(conclusionType string, ticketID, repo, body string) (*Conclusion, error) {
	if body == "" {
		return nil, &ValidationError{Field: "body", Message: "cannot be empty"}
	}

	ct := ConclusionType(conclusionType)
	if ct != TypeArchitect && ct != TypeWork && ct != TypeResearch {
		ct = TypeWork
	}

	now := time.Now().UTC()
	c := &Conclusion{
		ConclusionMeta: ConclusionMeta{
			ID:      uuid.New().String(),
			Type:    ct,
			Ticket:  ticketID,
			Repo:    repo,
			Created: now,
		},
		Body: body,
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Use a slug based on the ticket ID or "session" if no ticket
	slugSrc := "session"
	if ticketID != "" {
		slugSrc = ticketID
		if len(slugSrc) > 20 {
			slugSrc = slugSrc[:20]
		}
	}

	dirName := storage.DirName(slugSrc, c.ID, "session")
	entityDir := filepath.Join(s.sessionsDir, dirName)

	if err := os.MkdirAll(entityDir, 0755); err != nil {
		return nil, fmt.Errorf("create entity dir: %w", err)
	}

	data, err := storage.SerializeFrontmatter(&c.ConclusionMeta, c.Body)
	if err != nil {
		return nil, fmt.Errorf("serialize conclusion: %w", err)
	}

	target := filepath.Join(entityDir, "index.md")
	if err := storage.AtomicWriteFile(target, data); err != nil {
		return nil, fmt.Errorf("write conclusion: %w", err)
	}

	s.emit(events.ConclusionCreated, c.ID)
	return c, nil
}

// Get retrieves a conclusion by ID.
func (s *Store) Get(id string) (*Conclusion, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entityDir, err := s.findEntityDir(id)
	if err != nil {
		return nil, err
	}

	return s.loadIndex(entityDir)
}

// List returns all conclusions sorted by created time (newest first).
func (s *Store) List() ([]*Conclusion, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entries, err := os.ReadDir(s.sessionsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []*Conclusion{}, nil
		}
		return nil, fmt.Errorf("read sessions directory: %w", err)
	}

	var conclusions []*Conclusion
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		entityDir := filepath.Join(s.sessionsDir, entry.Name())
		c, err := s.loadIndex(entityDir)
		if err != nil {
			continue // skip broken entries
		}
		conclusions = append(conclusions, c)
	}

	// Sort by created descending
	for i := 0; i < len(conclusions); i++ {
		for j := i + 1; j < len(conclusions); j++ {
			if conclusions[j].Created.After(conclusions[i].Created) {
				conclusions[i], conclusions[j] = conclusions[j], conclusions[i]
			}
		}
	}

	return conclusions, nil
}

// loadIndex reads and parses index.md from the given entity directory.
func (s *Store) loadIndex(entityDir string) (*Conclusion, error) {
	data, err := os.ReadFile(filepath.Join(entityDir, "index.md"))
	if err != nil {
		return nil, fmt.Errorf("read index.md: %w", err)
	}

	meta, body, err := storage.ParseFrontmatter[ConclusionMeta](data)
	if err != nil {
		return nil, fmt.Errorf("parse index.md: %w", err)
	}

	return &Conclusion{
		ConclusionMeta: *meta,
		Body:           body,
	}, nil
}

// findEntityDir finds the entity directory for a conclusion.
func (s *Store) findEntityDir(id string) (string, error) {
	entries, err := os.ReadDir(s.sessionsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return "", &NotFoundError{Resource: "conclusion", ID: id}
		}
		return "", fmt.Errorf("read directory: %w", err)
	}

	shortID := storage.ShortID(id)

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, "-"+shortID) || strings.HasSuffix(name, "-"+id) {
			return filepath.Join(s.sessionsDir, name), nil
		}
	}

	return "", &NotFoundError{Resource: "conclusion", ID: id}
}
