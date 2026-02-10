package ticket

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

// Store manages ticket storage with directory-per-entity organized by status.
type Store struct {
	ticketsDir  string
	locks       sync.Map // maps ticket ID → *sync.Mutex
	bus         *events.Bus
	projectPath string
}

// ticketMu returns the mutex for a given ticket ID, creating one if needed.
func (s *Store) ticketMu(id string) *sync.Mutex {
	v, _ := s.locks.LoadOrStore(id, &sync.Mutex{})
	return v.(*sync.Mutex)
}

func (s *Store) emit(eventType events.EventType, ticketID string, payload any) {
	if s.bus == nil {
		return
	}
	s.bus.Emit(events.Event{
		Type:        eventType,
		ProjectPath: s.projectPath,
		TicketID:    ticketID,
		Payload:     payload,
	})
}

// NewStore creates a new Store and ensures the directory structure exists.
// bus and projectPath are optional; pass nil/"" to disable event emission.
func NewStore(ticketsDir string, bus *events.Bus, projectPath string) (*Store, error) {
	s := &Store{ticketsDir: ticketsDir, bus: bus, projectPath: projectPath}

	for _, status := range []Status{StatusBacklog, StatusProgress, StatusReview, StatusDone} {
		dir := filepath.Join(ticketsDir, string(status))
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("create directory %s: %w", dir, err)
		}
	}

	return s, nil
}

// Create creates a new ticket in the backlog.
func (s *Store) Create(title, body, ticketType string, dueDate *time.Time, references, tags []string) (*Ticket, error) {
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
			Tags:       tags,
			References: references,
			Due:        dueDate,
			Created:    now,
			Updated:    now,
		},
		Body:     body,
		Comments: []Comment{},
	}

	mu := s.ticketMu(ticket.ID)
	mu.Lock()
	defer mu.Unlock()

	if err := s.saveTicket(ticket, StatusBacklog); err != nil {
		return nil, fmt.Errorf("save ticket: %w", err)
	}

	s.emit(events.TicketCreated, ticket.ID, nil)
	return ticket, nil
}

// Get retrieves a ticket by ID, searching all status directories.
func (s *Store) Get(id string) (*Ticket, Status, error) {
	for _, status := range []Status{StatusBacklog, StatusProgress, StatusReview, StatusDone} {
		entityDir, err := s.findEntityDir(id, status)
		if err != nil {
			if storage.IsNotFound(err) {
				continue
			}
			return nil, "", err
		}

		ticket, err := s.loadIndex(entityDir)
		if err != nil {
			return nil, "", err
		}

		comments, err := storage.ListComments(entityDir)
		if err != nil {
			return nil, "", fmt.Errorf("load comments: %w", err)
		}
		ticket.Comments = comments

		return ticket, status, nil
	}
	return nil, "", &NotFoundError{Resource: "ticket", ID: id}
}

// Update modifies a ticket's title, body, references, and/or tags.
func (s *Store) Update(id string, title, body *string, references, tags *[]string) (*Ticket, error) {
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
	if tags != nil {
		ticket.Tags = *tags
	}

	ticket.Updated = time.Now().UTC()

	if titleChanged {
		// Title change means slug changes → rename directory
		newDirName := storage.DirName(ticket.Title, ticket.ID, "ticket")
		newDir := filepath.Join(s.ticketsDir, string(status), newDirName)
		if err := os.Rename(entityDir, newDir); err != nil {
			return nil, fmt.Errorf("rename entity dir: %w", err)
		}
		entityDir = newDir
	}

	if err := s.writeIndex(entityDir, ticket); err != nil {
		return nil, fmt.Errorf("save ticket: %w", err)
	}

	s.emit(events.TicketUpdated, ticket.ID, nil)
	return ticket, nil
}

// SetDueDate sets or clears the due date for a ticket.
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

	s.emit(events.TicketUpdated, ticket.ID, nil)
	return ticket, nil
}

// ClearDueDate removes the due date from a ticket.
func (s *Store) ClearDueDate(id string) (*Ticket, error) {
	return s.SetDueDate(id, nil)
}

// Delete removes a ticket (entire entity directory).
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
	s.emit(events.TicketDeleted, id, nil)
	return nil
}

// List returns all tickets with the given status (without loading comments).
func (s *Store) List(status Status) ([]*Ticket, error) {
	dir := filepath.Join(s.ticketsDir, string(status))
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []*Ticket{}, nil
		}
		return nil, fmt.Errorf("read directory: %w", err)
	}

	var tickets []*Ticket
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}

		entityDir := filepath.Join(dir, entry.Name())
		ticket, err := s.loadIndex(entityDir)
		if err != nil {
			return nil, err
		}
		tickets = append(tickets, ticket)
	}

	return tickets, nil
}

// ListAll returns all tickets grouped by status (without loading comments).
func (s *Store) ListAll() (map[Status][]*Ticket, error) {
	result := make(map[Status][]*Ticket)

	for _, status := range []Status{StatusBacklog, StatusProgress, StatusReview, StatusDone} {
		tickets, err := s.List(status)
		if err != nil {
			return nil, err
		}
		result[status] = tickets
	}

	return result, nil
}

// Move moves a ticket to a different status.
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

	// Ensure target status dir exists
	toDir := filepath.Join(s.ticketsDir, string(to))

	// Move entity directory to new status
	dirName := filepath.Base(entityDir)
	newDir := filepath.Join(toDir, dirName)
	if err := os.Rename(entityDir, newDir); err != nil {
		return fmt.Errorf("move entity dir: %w", err)
	}

	// Update index.md with new timestamp
	if err := s.writeIndex(newDir, ticket); err != nil {
		return fmt.Errorf("save ticket: %w", err)
	}

	s.emit(events.TicketMoved, ticket.ID, nil)
	return nil
}

// AddComment adds a comment to a ticket.
func (s *Store) AddComment(ticketID, author string, commentType CommentType, content string, action *storage.CommentAction) (*Comment, error) {
	mu := s.ticketMu(ticketID)
	mu.Lock()
	defer mu.Unlock()

	entityDir, _, err := s.findEntityDirAllStatuses(ticketID)
	if err != nil {
		return nil, err
	}

	comment, err := storage.CreateComment(entityDir, author, commentType, content, action)
	if err != nil {
		return nil, err
	}

	// Update the ticket's updated timestamp
	ticket, err := s.loadIndex(entityDir)
	if err != nil {
		return nil, err
	}
	ticket.Updated = time.Now().UTC()
	if err := s.writeIndex(entityDir, ticket); err != nil {
		return nil, fmt.Errorf("save ticket: %w", err)
	}

	s.emit(events.CommentAdded, ticketID, nil)
	return comment, nil
}

// ListComments returns all comments for a ticket sorted by created time.
func (s *Store) ListComments(ticketID string) ([]Comment, error) {
	entityDir, _, err := s.findEntityDirAllStatuses(ticketID)
	if err != nil {
		return nil, err
	}
	return storage.ListComments(entityDir)
}

// saveTicket creates the entity directory and writes index.md.
func (s *Store) saveTicket(ticket *Ticket, status Status) error {
	dirName := storage.DirName(ticket.Title, ticket.ID, "ticket")
	entityDir := filepath.Join(s.ticketsDir, string(status), dirName)

	if err := os.MkdirAll(entityDir, 0755); err != nil {
		return fmt.Errorf("create entity dir: %w", err)
	}

	return s.writeIndex(entityDir, ticket)
}

// writeIndex writes the index.md file in the given entity directory.
func (s *Store) writeIndex(entityDir string, ticket *Ticket) error {
	data, err := storage.SerializeFrontmatter(&ticket.TicketMeta, ticket.Body)
	if err != nil {
		return fmt.Errorf("serialize ticket: %w", err)
	}

	target := filepath.Join(entityDir, "index.md")
	return storage.AtomicWriteFile(target, data)
}

// loadIndex reads and parses index.md from the given entity directory.
func (s *Store) loadIndex(entityDir string) (*Ticket, error) {
	data, err := os.ReadFile(filepath.Join(entityDir, "index.md"))
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
		Comments:   []Comment{},
	}, nil
}

// findEntityDir finds the entity directory for a ticket in a specific status directory.
func (s *Store) findEntityDir(id string, status Status) (string, error) {
	dir := filepath.Join(s.ticketsDir, string(status))
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return "", &NotFoundError{Resource: "ticket", ID: id}
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
			return filepath.Join(dir, name), nil
		}
	}

	return "", &NotFoundError{Resource: "ticket", ID: id}
}

// findEntityDirAllStatuses searches all status directories for a ticket's entity directory.
func (s *Store) findEntityDirAllStatuses(id string) (string, Status, error) {
	for _, status := range []Status{StatusBacklog, StatusProgress, StatusReview, StatusDone} {
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

// IndexPath returns the filesystem path to a ticket's index.md file.
func (s *Store) IndexPath(id string) (string, error) {
	entityDir, _, err := s.findEntityDirAllStatuses(id)
	if err != nil {
		return "", err
	}
	return filepath.Join(entityDir, "index.md"), nil
}
