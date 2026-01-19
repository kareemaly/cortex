package ticket

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Store manages ticket storage with JSON files organized by status.
type Store struct {
	ticketsDir string
}

// NewStore creates a new Store and ensures the directory structure exists.
func NewStore(ticketsDir string) (*Store, error) {
	s := &Store{ticketsDir: ticketsDir}

	// Create status directories
	for _, status := range []Status{StatusBacklog, StatusProgress, StatusDone} {
		dir := filepath.Join(ticketsDir, string(status))
		if err := os.MkdirAll(dir, 0755); err != nil {
			return nil, fmt.Errorf("create directory %s: %w", dir, err)
		}
	}

	return s, nil
}

// Create creates a new ticket in the backlog.
func (s *Store) Create(title, body string) (*Ticket, error) {
	if title == "" {
		return nil, &ValidationError{Field: "title", Message: "cannot be empty"}
	}

	now := time.Now().UTC()
	ticket := &Ticket{
		ID:    uuid.New().String(),
		Title: title,
		Body:  body,
		Dates: Dates{
			Created:  now,
			Updated:  now,
			Approved: nil,
		},
		Sessions: []Session{},
	}

	if err := s.save(ticket, StatusBacklog); err != nil {
		return nil, fmt.Errorf("save ticket: %w", err)
	}

	return ticket, nil
}

// Get retrieves a ticket by ID, searching all status directories.
func (s *Store) Get(id string) (*Ticket, Status, error) {
	for _, status := range []Status{StatusBacklog, StatusProgress, StatusDone} {
		ticket, err := s.getFromStatus(id, status)
		if err == nil {
			return ticket, status, nil
		}
		if !IsNotFound(err) {
			return nil, "", err
		}
	}
	return nil, "", &NotFoundError{Resource: "ticket", ID: id}
}

// Update modifies a ticket's title and/or body.
func (s *Store) Update(id string, title, body *string) (*Ticket, error) {
	ticket, status, err := s.Get(id)
	if err != nil {
		return nil, err
	}

	if title != nil {
		if *title == "" {
			return nil, &ValidationError{Field: "title", Message: "cannot be empty"}
		}
		ticket.Title = *title
	}
	if body != nil {
		ticket.Body = *body
	}

	ticket.Dates.Updated = time.Now().UTC()

	if err := s.save(ticket, status); err != nil {
		return nil, fmt.Errorf("save ticket: %w", err)
	}

	return ticket, nil
}

// Delete removes a ticket.
func (s *Store) Delete(id string) error {
	for _, status := range []Status{StatusBacklog, StatusProgress, StatusDone} {
		path, err := s.findTicketPath(id, status)
		if err != nil {
			if IsNotFound(err) {
				continue
			}
			return err
		}
		if err := os.Remove(path); err != nil {
			return fmt.Errorf("remove ticket file: %w", err)
		}
		return nil
	}
	return &NotFoundError{Resource: "ticket", ID: id}
}

// List returns all tickets with the given status.
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
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		ticket, err := s.loadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, err
		}
		tickets = append(tickets, ticket)
	}

	return tickets, nil
}

// ListAll returns all tickets grouped by status.
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

// Move moves a ticket to a different status.
func (s *Store) Move(id string, to Status) error {
	ticket, from, err := s.Get(id)
	if err != nil {
		return err
	}

	if from == to {
		return nil
	}

	// Remove from old location
	oldPath, err := s.findTicketPath(id, from)
	if err != nil {
		return err
	}
	if err := os.Remove(oldPath); err != nil {
		return fmt.Errorf("remove old ticket file: %w", err)
	}

	// Set approved date when moving to done
	if to == StatusDone && ticket.Dates.Approved == nil {
		now := time.Now().UTC()
		ticket.Dates.Approved = &now
	}

	ticket.Dates.Updated = time.Now().UTC()

	// Save to new location
	if err := s.save(ticket, to); err != nil {
		return fmt.Errorf("save ticket: %w", err)
	}

	return nil
}

// AddSession adds a new session to a ticket.
func (s *Store) AddSession(ticketID, agent, tmuxWindow string, gitBase map[string]string) (*Session, error) {
	ticket, status, err := s.Get(ticketID)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	session := Session{
		ID:         uuid.New().String(),
		StartedAt:  now,
		EndedAt:    nil,
		Agent:      agent,
		TmuxWindow: tmuxWindow,
		GitBase:    gitBase,
		Report: Report{
			Files:     []string{},
			Decisions: []string{},
		},
		CurrentStatus: &StatusEntry{
			Status: AgentStatusStarting,
			At:     now,
		},
		StatusHistory: []StatusEntry{
			{Status: AgentStatusStarting, At: now},
		},
	}

	ticket.Sessions = append(ticket.Sessions, session)
	ticket.Dates.Updated = now

	if err := s.save(ticket, status); err != nil {
		return nil, fmt.Errorf("save ticket: %w", err)
	}

	return &session, nil
}

// EndSession marks a session as ended.
func (s *Store) EndSession(ticketID, sessionID string) error {
	ticket, status, err := s.Get(ticketID)
	if err != nil {
		return err
	}

	found := false
	now := time.Now().UTC()
	for i := range ticket.Sessions {
		if ticket.Sessions[i].ID == sessionID {
			ticket.Sessions[i].EndedAt = &now
			found = true
			break
		}
	}

	if !found {
		return &NotFoundError{Resource: "session", ID: sessionID}
	}

	ticket.Dates.Updated = now

	if err := s.save(ticket, status); err != nil {
		return fmt.Errorf("save ticket: %w", err)
	}

	return nil
}

// UpdateSessionStatus updates the current status of a session.
func (s *Store) UpdateSessionStatus(ticketID, sessionID string, agentStatus AgentStatus, tool, work *string) error {
	ticket, status, err := s.Get(ticketID)
	if err != nil {
		return err
	}

	now := time.Now().UTC()
	found := false
	for i := range ticket.Sessions {
		if ticket.Sessions[i].ID == sessionID {
			entry := StatusEntry{
				Status: agentStatus,
				Tool:   tool,
				Work:   work,
				At:     now,
			}
			ticket.Sessions[i].CurrentStatus = &entry
			ticket.Sessions[i].StatusHistory = append(ticket.Sessions[i].StatusHistory, entry)
			found = true
			break
		}
	}

	if !found {
		return &NotFoundError{Resource: "session", ID: sessionID}
	}

	ticket.Dates.Updated = now

	if err := s.save(ticket, status); err != nil {
		return fmt.Errorf("save ticket: %w", err)
	}

	return nil
}

// UpdateSessionReport updates the report of a session.
func (s *Store) UpdateSessionReport(ticketID, sessionID string, report Report) error {
	ticket, status, err := s.Get(ticketID)
	if err != nil {
		return err
	}

	found := false
	for i := range ticket.Sessions {
		if ticket.Sessions[i].ID == sessionID {
			ticket.Sessions[i].Report = report
			found = true
			break
		}
	}

	if !found {
		return &NotFoundError{Resource: "session", ID: sessionID}
	}

	ticket.Dates.Updated = time.Now().UTC()

	if err := s.save(ticket, status); err != nil {
		return fmt.Errorf("save ticket: %w", err)
	}

	return nil
}

// filename generates the filename for a ticket: {slug}-{id}.json
func (s *Store) filename(ticket *Ticket) string {
	slug := GenerateSlug(ticket.Title)
	// Use first 8 chars of UUID for readability
	shortID := ticket.ID
	if len(shortID) > 8 {
		shortID = shortID[:8]
	}
	return fmt.Sprintf("%s-%s.json", slug, shortID)
}

// save writes a ticket to the appropriate status directory.
func (s *Store) save(ticket *Ticket, status Status) error {
	dir := filepath.Join(s.ticketsDir, string(status))
	path := filepath.Join(dir, s.filename(ticket))

	data, err := json.MarshalIndent(ticket, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal ticket: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("write file: %w", err)
	}

	return nil
}

// loadFile reads and unmarshals a ticket from a file.
func (s *Store) loadFile(path string) (*Ticket, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read file: %w", err)
	}

	var ticket Ticket
	if err := json.Unmarshal(data, &ticket); err != nil {
		return nil, fmt.Errorf("unmarshal ticket: %w", err)
	}

	return &ticket, nil
}

// getFromStatus retrieves a ticket from a specific status directory.
func (s *Store) getFromStatus(id string, status Status) (*Ticket, error) {
	path, err := s.findTicketPath(id, status)
	if err != nil {
		return nil, err
	}
	return s.loadFile(path)
}

// findTicketPath finds the file path for a ticket in a status directory.
func (s *Store) findTicketPath(id string, status Status) (string, error) {
	dir := filepath.Join(s.ticketsDir, string(status))
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return "", &NotFoundError{Resource: "ticket", ID: id}
		}
		return "", fmt.Errorf("read directory: %w", err)
	}

	// Look for file ending with -{id}.json or -{shortID}.json
	shortID := id
	if len(shortID) > 8 {
		shortID = shortID[:8]
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.HasSuffix(name, "-"+shortID+".json") || strings.HasSuffix(name, "-"+id+".json") {
			return filepath.Join(dir, name), nil
		}
	}

	return "", &NotFoundError{Resource: "ticket", ID: id}
}
