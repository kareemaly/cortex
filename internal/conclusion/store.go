package conclusion

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/kareemaly/cortex/internal/entity"
	"github.com/kareemaly/cortex/internal/events"
	"github.com/kareemaly/cortex/internal/storage"
)

type Store struct {
	*entity.BaseStore
	mu sync.RWMutex
}

func NewStore(sessionsDir string, bus *events.Bus, projectPath string) (*Store, error) {
	base, err := entity.NewBaseStore(sessionsDir, bus, projectPath)
	if err != nil {
		return nil, err
	}

	return &Store{BaseStore: base}, nil
}

func (s *Store) Create(conclusionType string, ticketID, repo, body string, startedAt time.Time, prompt string) (*Conclusion, error) {
	if body == "" {
		return nil, &ValidationError{Field: "body", Message: "cannot be empty"}
	}

	ct := ConclusionType(conclusionType)
	if ct != TypeArchitect && ct != TypeWork && ct != TypeResearch && ct != TypeCollab {
		ct = TypeWork
	}

	now := time.Now().UTC()
	c := &Conclusion{
		ConclusionMeta: ConclusionMeta{
			ID:          uuid.New().String(),
			Type:        ct,
			Ticket:      ticketID,
			Repo:        repo,
			Prompt:      prompt,
			ConcludedAt: now,
			StartedAt:   startedAt,
		},
		Body: body,
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	slugSrc := "session"
	if ticketID != "" {
		slugSrc = ticketID
		if len(slugSrc) > 20 {
			slugSrc = slugSrc[:20]
		}
	}

	dirName := storage.DirName(slugSrc, c.ID, "session")
	entityDir := filepath.Join(s.RootDir(), dirName)

	if err := os.MkdirAll(entityDir, 0755); err != nil {
		return nil, fmt.Errorf("create entity dir: %w", err)
	}

	data, err := storage.SerializeFrontmatter(&c.ConclusionMeta, c.Body)
	if err != nil {
		return nil, fmt.Errorf("serialize conclusion: %w", err)
	}

	if err := s.WriteIndexBytes(entityDir, data); err != nil {
		return nil, fmt.Errorf("write conclusion: %w", err)
	}

	s.Emit(events.ConclusionCreated, c.ID, nil)
	return c, nil
}

func (s *Store) IndexPath(id string) (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	entityDir, err := s.FindEntityDir("conclusion", id)
	if err != nil {
		return "", err
	}
	return filepath.Join(entityDir, "index.md"), nil
}

func (s *Store) Get(id string) (*Conclusion, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entityDir, err := s.FindEntityDir("conclusion", id)
	if err != nil {
		return nil, err
	}

	return s.loadIndex(entityDir)
}

type ListOptions struct {
	Type   string
	Limit  int
	Offset int
}

func (s *Store) ListWithOptions(opts ListOptions) ([]*Conclusion, int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	entityDirs, err := s.ListEntries(s.RootDir())
	if err != nil {
		return nil, 0, err
	}

	var all []*Conclusion
	for _, entityDir := range entityDirs {
		c, err := s.loadIndex(entityDir)
		if err != nil {
			continue
		}
		all = append(all, c)
	}

	for i := 0; i < len(all); i++ {
		for j := i + 1; j < len(all); j++ {
			if all[j].ConcludedAt.After(all[i].ConcludedAt) {
				all[i], all[j] = all[j], all[i]
			}
		}
	}

	var filtered []*Conclusion
	if opts.Type == "" {
		filtered = all
	} else {
		for _, c := range all {
			if string(c.Type) == opts.Type {
				filtered = append(filtered, c)
			}
		}
	}

	total := len(filtered)

	if opts.Offset >= total {
		return []*Conclusion{}, total, nil
	}
	filtered = filtered[opts.Offset:]
	if opts.Limit > 0 && len(filtered) > opts.Limit {
		filtered = filtered[:opts.Limit]
	}

	return filtered, total, nil
}

func (s *Store) List() ([]*Conclusion, error) {
	items, _, err := s.ListWithOptions(ListOptions{})
	return items, err
}

func (s *Store) loadIndex(entityDir string) (*Conclusion, error) {
	data, err := s.LoadIndexBytes(entityDir)
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
