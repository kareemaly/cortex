package ticket

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/kareemaly/cortex/internal/events"
)

// Store manages ticket storage with JSON files organized by status.
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

	// Create status directories
	for _, status := range []Status{StatusBacklog, StatusProgress, StatusReview, StatusDone} {
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
			Created: now,
			Updated: now,
		},
		Comments: []Comment{},
		Session:  nil,
	}

	mu := s.ticketMu(ticket.ID)
	mu.Lock()
	defer mu.Unlock()

	if err := s.save(ticket, StatusBacklog); err != nil {
		return nil, fmt.Errorf("save ticket: %w", err)
	}

	s.emit(events.TicketCreated, ticket.ID, nil)
	return ticket, nil
}

// Get retrieves a ticket by ID, searching all status directories.
func (s *Store) Get(id string) (*Ticket, Status, error) {
	for _, status := range []Status{StatusBacklog, StatusProgress, StatusReview, StatusDone} {
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
	mu := s.ticketMu(id)
	mu.Lock()
	defer mu.Unlock()

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

	s.emit(events.TicketUpdated, ticket.ID, nil)
	return ticket, nil
}

// Delete removes a ticket.
func (s *Store) Delete(id string) error {
	mu := s.ticketMu(id)
	mu.Lock()
	defer mu.Unlock()

	for _, status := range []Status{StatusBacklog, StatusProgress, StatusReview, StatusDone} {
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
		s.locks.Delete(id)
		s.emit(events.TicketDeleted, id, nil)
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
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".json") || strings.HasPrefix(name, ".tmp-") {
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

	// Set date fields based on target status
	now := time.Now().UTC()
	switch to {
	case StatusProgress:
		if ticket.Dates.Progress == nil {
			ticket.Dates.Progress = &now
		}
	case StatusReview:
		ticket.Dates.Reviewed = &now
	case StatusDone:
		ticket.Dates.Done = &now
	}

	ticket.Dates.Updated = now

	// Save to new location
	if err := s.save(ticket, to); err != nil {
		return fmt.Errorf("save ticket: %w", err)
	}

	s.emit(events.TicketMoved, ticket.ID, nil)
	return nil
}

// SetSession sets the session for a ticket (replaces any existing session).
func (s *Store) SetSession(ticketID, agent, tmuxWindow string, worktreePath, featureBranch *string) (*Session, error) {
	mu := s.ticketMu(ticketID)
	mu.Lock()
	defer mu.Unlock()

	ticket, status, err := s.Get(ticketID)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	session := &Session{
		ID:            uuid.New().String(),
		StartedAt:     now,
		EndedAt:       nil,
		Agent:         agent,
		TmuxWindow:    tmuxWindow,
		WorktreePath:  worktreePath,
		FeatureBranch: featureBranch,
		CurrentStatus: &StatusEntry{
			Status: AgentStatusStarting,
			At:     now,
		},
		StatusHistory: []StatusEntry{
			{Status: AgentStatusStarting, At: now},
		},
	}

	ticket.Session = session
	ticket.Dates.Updated = now

	if err := s.save(ticket, status); err != nil {
		return nil, fmt.Errorf("save ticket: %w", err)
	}

	s.emit(events.SessionStarted, ticketID, nil)
	return session, nil
}

// EndSession marks the ticket's session as ended.
func (s *Store) EndSession(ticketID string) error {
	mu := s.ticketMu(ticketID)
	mu.Lock()
	defer mu.Unlock()

	ticket, status, err := s.Get(ticketID)
	if err != nil {
		return err
	}

	if ticket.Session == nil {
		return &NotFoundError{Resource: "session", ID: ticketID}
	}

	now := time.Now().UTC()
	ticket.Session.EndedAt = &now
	ticket.Dates.Updated = now

	if err := s.save(ticket, status); err != nil {
		return fmt.Errorf("save ticket: %w", err)
	}

	s.emit(events.SessionEnded, ticketID, nil)
	return nil
}

// UpdateSessionStatus updates the current status of the ticket's session.
func (s *Store) UpdateSessionStatus(ticketID string, agentStatus AgentStatus, tool, work *string) error {
	mu := s.ticketMu(ticketID)
	mu.Lock()
	defer mu.Unlock()

	ticket, status, err := s.Get(ticketID)
	if err != nil {
		return err
	}

	if ticket.Session == nil {
		return &NotFoundError{Resource: "session", ID: ticketID}
	}

	now := time.Now().UTC()
	entry := StatusEntry{
		Status: agentStatus,
		Tool:   tool,
		Work:   work,
		At:     now,
	}
	ticket.Session.CurrentStatus = &entry
	ticket.Session.StatusHistory = append(ticket.Session.StatusHistory, entry)
	ticket.Dates.Updated = now

	if err := s.save(ticket, status); err != nil {
		return fmt.Errorf("save ticket: %w", err)
	}

	s.emit(events.SessionStatus, ticketID, nil)
	return nil
}

// AddComment adds a comment to a ticket.
func (s *Store) AddComment(ticketID, sessionID string, commentType CommentType, content string) (*Comment, error) {
	mu := s.ticketMu(ticketID)
	mu.Lock()
	defer mu.Unlock()

	ticket, status, err := s.Get(ticketID)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	comment := Comment{
		ID:        uuid.New().String(),
		SessionID: sessionID,
		Type:      commentType,
		Content:   content,
		CreatedAt: now,
	}

	ticket.Comments = append(ticket.Comments, comment)
	ticket.Dates.Updated = now

	if err := s.save(ticket, status); err != nil {
		return nil, fmt.Errorf("save ticket: %w", err)
	}

	s.emit(events.CommentAdded, ticketID, nil)
	return &comment, nil
}

// AddReviewRequest adds a review request to the ticket's active session.
// Returns the total number of review requests after adding.
func (s *Store) AddReviewRequest(ticketID, repoPath, summary string) (int, error) {
	mu := s.ticketMu(ticketID)
	mu.Lock()
	defer mu.Unlock()

	ticket, status, err := s.Get(ticketID)
	if err != nil {
		return 0, err
	}

	if ticket.Session == nil {
		return 0, &NotFoundError{Resource: "session", ID: ticketID}
	}

	if !ticket.Session.IsActive() {
		return 0, &ValidationError{Field: "session", Message: "session is not active"}
	}

	now := time.Now().UTC()
	review := ReviewRequest{
		RepoPath:    repoPath,
		Summary:     summary,
		RequestedAt: now,
	}

	ticket.Session.RequestedReviews = append(ticket.Session.RequestedReviews, review)
	ticket.Dates.Updated = now

	if err := s.save(ticket, status); err != nil {
		return 0, fmt.Errorf("save ticket: %w", err)
	}

	s.emit(events.ReviewRequested, ticketID, nil)
	return len(ticket.Session.RequestedReviews), nil
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

// save writes a ticket to the appropriate status directory using atomic write.
// It writes to a temp file first, then renames to the target path to prevent
// partial writes from corrupting data.
func (s *Store) save(ticket *Ticket, status Status) error {
	dir := filepath.Join(s.ticketsDir, string(status))
	target := filepath.Join(dir, s.filename(ticket))

	data, err := json.MarshalIndent(ticket, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal ticket: %w", err)
	}

	// Write to a temp file in the same directory (same filesystem for atomic rename)
	tmp, err := os.CreateTemp(dir, ".tmp-*.json")
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
