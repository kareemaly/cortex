package ticket

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
	locks sync.Map
}

func (s *Store) ticketMu(id string) *sync.Mutex {
	v, _ := s.locks.LoadOrStore(id, &sync.Mutex{})
	return v.(*sync.Mutex)
}

func NewStore(ticketsDir string, bus *events.Bus, projectPath string) (*Store, error) {
	base, err := entity.NewBaseStore(ticketsDir, bus, projectPath)
	if err != nil {
		return nil, err
	}

	s := &Store{BaseStore: base}

	for _, status := range []Status{StatusBacklog, StatusProgress, StatusDone} {
		dir := filepath.Join(ticketsDir, string(status))
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("create directory %s: %w", dir, err)
		}
	}

	return s, nil
}

func (s *Store) Create(title, body, ticketType string, dueDate *time.Time, references []string, repo, path string) (*Ticket, error) {
	if title == "" {
		return nil, &ValidationError{Field: "title", Message: "cannot be empty"}
	}

	if ticketType == "" {
		ticketType = DefaultTicketType
	}

	now := time.Now().UTC()
	ticket := &Ticket{
		TicketMeta: TicketMeta{
			ID:         uuid.New().String(),
			Title:      title,
			Type:       ticketType,
			Repo:       repo,
			Path:       path,
			References: references,
			Due:        dueDate,
			Created:    now,
			Updated:    now,
		},
		Body: body,
	}

	mu := s.ticketMu(ticket.ID)
	mu.Lock()
	defer mu.Unlock()

	if err := s.saveTicket(ticket, StatusBacklog); err != nil {
		return nil, fmt.Errorf("save ticket: %w", err)
	}

	s.Emit(events.TicketCreated, ticket.ID, nil)
	return ticket, nil
}

func (s *Store) Get(id string) (*Ticket, Status, error) {
	entityDir, err := s.findEntityDir(id, StatusBacklog)
	if err == nil {
		ticket, err := s.loadIndex(entityDir)
		if err != nil {
			return nil, "", err
		}
		return ticket, StatusBacklog, nil
	}

	entityDir, err = s.findEntityDir(id, StatusProgress)
	if err == nil {
		ticket, err := s.loadIndex(entityDir)
		if err != nil {
			return nil, "", err
		}
		return ticket, StatusProgress, nil
	}

	entityDir, err = s.findEntityDir(id, StatusDone)
	if err == nil {
		ticket, err := s.loadIndex(entityDir)
		if err != nil {
			return nil, "", err
		}
		return ticket, StatusDone, nil
	}

	return nil, "", &NotFoundError{Resource: "ticket", ID: id}
}

func (s *Store) Update(id string, title, body *string, references *[]string) (*Ticket, error) {
	mu := s.ticketMu(id)
	mu.Lock()
	defer mu.Unlock()

	entityDir, status, err := s.findEntityDirAllStatuses(id)
	if err != nil {
		return nil, err
	}

	ticket, err := s.loadIndex(entityDir)
	if err != nil {
		return nil, err
	}

	titleChanged := false
	if title != nil {
		if *title == "" {
			return nil, &ValidationError{Field: "title", Message: "cannot be empty"}
		}
		if ticket.Title != *title {
			titleChanged = true
		}
		ticket.Title = *title
	}
	if body != nil {
		ticket.Body = *body
	}
	if references != nil {
		ticket.References = *references
	}

	ticket.Updated = time.Now().UTC()

	if titleChanged {
		newDirName := storage.DirName(ticket.Title, ticket.ID, "ticket")
		newDir := filepath.Join(s.RootDir(), string(status), newDirName)
		if err := os.Rename(entityDir, newDir); err != nil {
			return nil, fmt.Errorf("rename entity dir: %w", err)
		}
		entityDir = newDir
	}

	if err := s.writeIndex(entityDir, ticket); err != nil {
		return nil, fmt.Errorf("save ticket: %w", err)
	}

	s.Emit(events.TicketUpdated, ticket.ID, nil)
	return ticket, nil
}

func (s *Store) SetSession(id string, sessionID string) (*Ticket, error) {
	mu := s.ticketMu(id)
	mu.Lock()
	defer mu.Unlock()

	entityDir, _, err := s.findEntityDirAllStatuses(id)
	if err != nil {
		return nil, err
	}

	ticket, err := s.loadIndex(entityDir)
	if err != nil {
		return nil, err
	}

	ticket.Session = sessionID
	ticket.Updated = time.Now().UTC()

	if err := s.writeIndex(entityDir, ticket); err != nil {
		return nil, fmt.Errorf("save ticket: %w", err)
	}

	s.Emit(events.TicketUpdated, ticket.ID, nil)
	return ticket, nil
}

func (s *Store) SetDueDate(id string, dueDate *time.Time) (*Ticket, error) {
	mu := s.ticketMu(id)
	mu.Lock()
	defer mu.Unlock()

	entityDir, _, err := s.findEntityDirAllStatuses(id)
	if err != nil {
		return nil, err
	}

	ticket, err := s.loadIndex(entityDir)
	if err != nil {
		return nil, err
	}

	ticket.Due = dueDate
	ticket.Updated = time.Now().UTC()

	if err := s.writeIndex(entityDir, ticket); err != nil {
		return nil, fmt.Errorf("save ticket: %w", err)
	}

	s.Emit(events.TicketUpdated, ticket.ID, nil)
	return ticket, nil
}

func (s *Store) ClearDueDate(id string) (*Ticket, error) {
	return s.SetDueDate(id, nil)
}

func (s *Store) Delete(id string) error {
	mu := s.ticketMu(id)
	mu.Lock()
	defer mu.Unlock()

	entityDir, _, err := s.findEntityDirAllStatuses(id)
	if err != nil {
		return err
	}

	if err := os.RemoveAll(entityDir); err != nil {
		return fmt.Errorf("remove entity directory: %w", err)
	}

	s.locks.Delete(id)
	s.Emit(events.TicketDeleted, id, nil)
	return nil
}

func (s *Store) List(status Status) ([]*Ticket, error) {
	dir := filepath.Join(s.RootDir(), string(status))
	entityDirs, err := s.ListEntries(dir)
	if err != nil {
		return nil, err
	}

	var tickets []*Ticket
	for _, entityDir := range entityDirs {
		ticket, err := s.loadIndex(entityDir)
		if err != nil {
			return nil, err
		}
		tickets = append(tickets, ticket)
	}

	return tickets, nil
}

func (s *Store) ListAll() (map[Status][]*Ticket, error) {
	result := make(map[Status][]*Ticket)

	for _, status := range []Status{StatusBacklog, StatusProgress, StatusDone} {
		tickets, err := s.List(status)
		if err != nil {
			return nil, err
		}
		result[status] = tickets
	}

	return result, nil
}

func (s *Store) Move(id string, to Status) error {
	mu := s.ticketMu(id)
	mu.Lock()
	defer mu.Unlock()

	entityDir, from, err := s.findEntityDirAllStatuses(id)
	if err != nil {
		return err
	}

	if from == to {
		return nil
	}

	ticket, err := s.loadIndex(entityDir)
	if err != nil {
		return err
	}

	ticket.Updated = time.Now().UTC()

	toDir := filepath.Join(s.RootDir(), string(to))

	dirName := filepath.Base(entityDir)
	newDir := filepath.Join(toDir, dirName)
	if err := os.Rename(entityDir, newDir); err != nil {
		return fmt.Errorf("move entity dir: %w", err)
	}

	if err := s.writeIndex(newDir, ticket); err != nil {
		return fmt.Errorf("save ticket: %w", err)
	}

	s.Emit(events.TicketMoved, ticket.ID, nil)
	return nil
}

func (s *Store) saveTicket(ticket *Ticket, status Status) error {
	dirName := storage.DirName(ticket.Title, ticket.ID, "ticket")
	entityDir := filepath.Join(s.RootDir(), string(status), dirName)

	if err := os.MkdirAll(entityDir, 0755); err != nil {
		return fmt.Errorf("create entity dir: %w", err)
	}

	return s.writeIndex(entityDir, ticket)
}

func (s *Store) writeIndex(entityDir string, ticket *Ticket) error {
	data, err := storage.SerializeFrontmatter(&ticket.TicketMeta, ticket.Body)
	if err != nil {
		return fmt.Errorf("serialize ticket: %w", err)
	}

	return s.WriteIndexBytes(entityDir, data)
}

func (s *Store) loadIndex(entityDir string) (*Ticket, error) {
	data, err := s.LoadIndexBytes(entityDir)
	if err != nil {
		return nil, fmt.Errorf("read index.md: %w", err)
	}

	meta, body, err := storage.ParseFrontmatter[TicketMeta](data)
	if err != nil {
		return nil, fmt.Errorf("parse index.md: %w", err)
	}

	return &Ticket{
		TicketMeta: *meta,
		Body:       body,
	}, nil
}

func (s *Store) findEntityDir(id string, status Status) (string, error) {
	return s.FindEntityDir("ticket", id, string(status))
}

func (s *Store) findEntityDirAllStatuses(id string) (string, Status, error) {
	for _, status := range []Status{StatusBacklog, StatusProgress, StatusDone} {
		entityDir, err := s.findEntityDir(id, status)
		if err == nil {
			return entityDir, status, nil
		}
		if !storage.IsNotFound(err) {
			return "", "", err
		}
	}
	return "", "", &NotFoundError{Resource: "ticket", ID: id}
}

func (s *Store) IndexPath(id string) (string, error) {
	entityDir, _, err := s.findEntityDirAllStatuses(id)
	if err != nil {
		return "", err
	}
	return filepath.Join(entityDir, "index.md"), nil
}
