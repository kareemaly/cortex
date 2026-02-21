package notes

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/kareemaly/cortex/internal/events"
	"github.com/kareemaly/cortex/internal/storage"
	"gopkg.in/yaml.v3"
)

const maxNotes = 50

// Store manages notes backed by a single YAML file.
type Store struct {
	path        string
	bus         *events.Bus
	projectPath string
	mu          sync.Mutex
}

// NewStore creates a new notes store backed by the given file path.
func NewStore(path string, bus *events.Bus, projectPath string) *Store {
	return &Store{
		path:        path,
		bus:         bus,
		projectPath: projectPath,
	}
}

// List returns all notes.
func (s *Store) List() ([]Note, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.load()
}

// Create adds a new note. Returns an error if the cap is reached.
func (s *Store) Create(text string, due *time.Time) (*Note, error) {
	if text == "" {
		return nil, &storage.ValidationError{Field: "text", Message: "cannot be empty"}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	notes, err := s.load()
	if err != nil {
		return nil, err
	}

	if len(notes) >= maxNotes {
		return nil, &storage.ValidationError{Field: "notes", Message: fmt.Sprintf("maximum of %d notes reached", maxNotes)}
	}

	note := Note{
		ID:      shortID(),
		Text:    text,
		Due:     due,
		Created: time.Now().UTC(),
	}

	notes = append(notes, note)
	if err := s.save(notes); err != nil {
		return nil, err
	}

	if s.bus != nil {
		s.bus.Emit(events.Event{
			Type:        events.NoteCreated,
			ProjectPath: s.projectPath,
			Payload:     note,
		})
	}

	return &note, nil
}

// Update modifies a note's text and/or due date.
func (s *Store) Update(id string, text *string, due *string) (*Note, error) {
	if id == "" {
		return nil, &storage.ValidationError{Field: "id", Message: "cannot be empty"}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	notes, err := s.load()
	if err != nil {
		return nil, err
	}

	idx := -1
	for i := range notes {
		if notes[i].ID == id {
			idx = i
			break
		}
	}
	if idx == -1 {
		return nil, &storage.NotFoundError{Resource: "note", ID: id}
	}

	if text != nil {
		if *text == "" {
			return nil, &storage.ValidationError{Field: "text", Message: "cannot be empty"}
		}
		notes[idx].Text = *text
	}
	if due != nil {
		if *due == "" {
			notes[idx].Due = nil
		} else {
			parsed, parseErr := time.Parse(time.DateOnly, *due)
			if parseErr != nil {
				return nil, &storage.ValidationError{Field: "due", Message: "must be YYYY-MM-DD format"}
			}
			notes[idx].Due = &parsed
		}
	}

	if err := s.save(notes); err != nil {
		return nil, err
	}

	if s.bus != nil {
		s.bus.Emit(events.Event{
			Type:        events.NoteUpdated,
			ProjectPath: s.projectPath,
			Payload:     notes[idx],
		})
	}

	return &notes[idx], nil
}

// Delete removes a note by ID.
func (s *Store) Delete(id string) error {
	if id == "" {
		return &storage.ValidationError{Field: "id", Message: "cannot be empty"}
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	notes, err := s.load()
	if err != nil {
		return err
	}

	idx := -1
	for i := range notes {
		if notes[i].ID == id {
			idx = i
			break
		}
	}
	if idx == -1 {
		return &storage.NotFoundError{Resource: "note", ID: id}
	}

	notes = append(notes[:idx], notes[idx+1:]...)
	if err := s.save(notes); err != nil {
		return err
	}

	if s.bus != nil {
		s.bus.Emit(events.Event{
			Type:        events.NoteDeleted,
			ProjectPath: s.projectPath,
			Payload:     map[string]string{"id": id},
		})
	}

	return nil
}

// load reads notes from the YAML file. Returns empty slice if file doesn't exist.
func (s *Store) load() ([]Note, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return []Note{}, nil
		}
		return nil, fmt.Errorf("read notes file: %w", err)
	}

	if len(data) == 0 {
		return []Note{}, nil
	}

	var notes []Note
	if err := yaml.Unmarshal(data, &notes); err != nil {
		return nil, fmt.Errorf("unmarshal notes: %w", err)
	}

	if notes == nil {
		notes = []Note{}
	}

	return notes, nil
}

// save writes notes to the YAML file atomically.
func (s *Store) save(notes []Note) error {
	data, err := yaml.Marshal(notes)
	if err != nil {
		return fmt.Errorf("marshal notes: %w", err)
	}

	return storage.AtomicWriteFile(s.path, data)
}

// shortID generates an 8-character short ID from a UUID.
func shortID() string {
	return uuid.New().String()[:8]
}
