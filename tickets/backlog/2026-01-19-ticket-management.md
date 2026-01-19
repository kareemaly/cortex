# Ticket Management

Implement the ticket file management layer for JSON-based ticket storage.

## Context

Tickets are JSON files stored in `.cortex/tickets/{backlog,progress,done}/`. This package provides CRUD operations and is the foundation for MCP tools and daemon API.

Reference DESIGN.md for the ticket JSON schema.

## Requirements

### 1. internal/ticket/ticket.go

Define the ticket data structures matching DESIGN.md schema:

```go
package ticket

import (
	"time"

	"github.com/google/uuid"
)

// Ticket represents a work item
type Ticket struct {
	ID       string   `json:"id"`
	Title    string   `json:"title"`
	Body     string   `json:"body"`
	Dates    Dates    `json:"dates"`
	Sessions []Session `json:"sessions"`
}

type Dates struct {
	Created  time.Time  `json:"created"`
	Updated  time.Time  `json:"updated"`
	Approved *time.Time `json:"approved,omitempty"`
}

type Session struct {
	ID            string         `json:"id"`
	StartedAt     time.Time      `json:"started_at"`
	EndedAt       *time.Time     `json:"ended_at,omitempty"`
	Agent         string         `json:"agent"`
	TmuxWindow    string         `json:"tmux_window"`
	GitBase       map[string]string `json:"git_base"`
	Report        *Report        `json:"report,omitempty"`
	CurrentStatus *Status        `json:"current_status,omitempty"`
	StatusHistory []Status       `json:"status_history"`
}

type Report struct {
	Files        []string `json:"files"`
	ScopeChanges *string  `json:"scope_changes,omitempty"`
	Decisions    []string `json:"decisions"`
	Summary      string   `json:"summary"`
}

type Status struct {
	Status string     `json:"status"` // starting, in_progress, idle, waiting_permission, error
	Tool   *string    `json:"tool,omitempty"`
	Work   *string    `json:"work,omitempty"`
	At     time.Time  `json:"at"`
}

// NewTicket creates a new ticket with generated ID and timestamps
func NewTicket(title, body string) *Ticket {
	now := time.Now().UTC()
	return &Ticket{
		ID:    uuid.New().String(),
		Title: title,
		Body:  body,
		Dates: Dates{
			Created: now,
			Updated: now,
		},
		Sessions: []Session{},
	}
}

// Slug returns a URL-safe slug from the title (max 20 chars)
func (t *Ticket) Slug() string {
	// Implementation: lowercase, replace spaces with hyphens,
	// remove non-alphanumeric, truncate to 20 chars
}

// AddSession adds a new session to the ticket
func (t *Ticket) AddSession(agent string) *Session {
	now := time.Now().UTC()
	sess := Session{
		ID:        uuid.New().String(),
		StartedAt: now,
		Agent:     agent,
		TmuxWindow: t.Slug(),
		GitBase:   make(map[string]string),
		StatusHistory: []Status{
			{Status: "starting", At: now},
		},
	}
	t.Sessions = append(t.Sessions, sess)
	t.Dates.Updated = now
	return &t.Sessions[len(t.Sessions)-1]
}

// ActiveSession returns the current active session (no EndedAt) or nil
func (t *Ticket) ActiveSession() *Session {
	for i := range t.Sessions {
		if t.Sessions[i].EndedAt == nil {
			return &t.Sessions[i]
		}
	}
	return nil
}
```

### 2. internal/ticket/store.go

File-based storage operations:

```go
package ticket

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Status represents ticket status (folder location)
type TicketStatus string

const (
	StatusBacklog  TicketStatus = "backlog"
	StatusProgress TicketStatus = "progress"
	StatusDone     TicketStatus = "done"
)

// Store manages ticket files in a project
type Store struct {
	projectPath string // Path to project root (contains .cortex/)
}

// NewStore creates a store for the given project path
func NewStore(projectPath string) *Store {
	return &Store{projectPath: projectPath}
}

// ticketsDir returns the path to .cortex/tickets/
func (s *Store) ticketsDir() string {
	return filepath.Join(s.projectPath, ".cortex", "tickets")
}

// statusDir returns the path to a status folder
func (s *Store) statusDir(status TicketStatus) string {
	return filepath.Join(s.ticketsDir(), string(status))
}

// ticketPath returns the path to a ticket file
func (s *Store) ticketPath(status TicketStatus, id string) string {
	return filepath.Join(s.statusDir(status), id+".json")
}

// Create saves a new ticket to backlog
func (s *Store) Create(t *Ticket) error {
	if err := s.ensureDir(StatusBacklog); err != nil {
		return err
	}
	return s.write(StatusBacklog, t)
}

// Read loads a ticket by ID, searching all status folders
func (s *Store) Read(id string) (*Ticket, TicketStatus, error) {
	for _, status := range []TicketStatus{StatusBacklog, StatusProgress, StatusDone} {
		path := s.ticketPath(status, id)
		if _, err := os.Stat(path); err == nil {
			t, err := s.readFile(path)
			if err != nil {
				return nil, "", err
			}
			return t, status, nil
		}
	}
	return nil, "", fmt.Errorf("ticket not found: %s", id)
}

// Update saves changes to an existing ticket
func (s *Store) Update(t *Ticket) error {
	_, status, err := s.Read(t.ID)
	if err != nil {
		return err
	}
	t.Dates.Updated = time.Now().UTC()
	return s.write(status, t)
}

// Delete removes a ticket
func (s *Store) Delete(id string) error {
	_, status, err := s.Read(id)
	if err != nil {
		return err
	}
	return os.Remove(s.ticketPath(status, id))
}

// List returns all tickets with the given status
func (s *Store) List(status TicketStatus) ([]*Ticket, error) {
	dir := s.statusDir(status)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []*Ticket{}, nil
		}
		return nil, err
	}

	var tickets []*Ticket
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		t, err := s.readFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, err
		}
		tickets = append(tickets, t)
	}
	return tickets, nil
}

// ListAll returns all tickets grouped by status
func (s *Store) ListAll() (map[TicketStatus][]*Ticket, error) {
	result := make(map[TicketStatus][]*Ticket)
	for _, status := range []TicketStatus{StatusBacklog, StatusProgress, StatusDone} {
		tickets, err := s.List(status)
		if err != nil {
			return nil, err
		}
		result[status] = tickets
	}
	return result, nil
}

// Move moves a ticket to a different status
func (s *Store) Move(id string, to TicketStatus) error {
	t, from, err := s.Read(id)
	if err != nil {
		return err
	}
	if from == to {
		return nil // Already in target status
	}

	// If moving to done, set approved timestamp
	if to == StatusDone && t.Dates.Approved == nil {
		now := time.Now().UTC()
		t.Dates.Approved = &now
	}

	// Ensure target dir exists
	if err := s.ensureDir(to); err != nil {
		return err
	}

	// Write to new location
	if err := s.write(to, t); err != nil {
		return err
	}

	// Remove from old location
	return os.Remove(s.ticketPath(from, id))
}

// FindByStatus returns the status of a ticket
func (s *Store) FindByStatus(id string) (TicketStatus, error) {
	_, status, err := s.Read(id)
	return status, err
}

// Helper methods

func (s *Store) ensureDir(status TicketStatus) error {
	return os.MkdirAll(s.statusDir(status), 0755)
}

func (s *Store) write(status TicketStatus, t *Ticket) error {
	data, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.ticketPath(status, t.ID), data, 0644)
}

func (s *Store) readFile(path string) (*Ticket, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var t Ticket
	if err := json.Unmarshal(data, &t); err != nil {
		return nil, err
	}
	return &t, nil
}
```

### 3. internal/ticket/slug.go

Slug generation utility:

```go
package ticket

import (
	"regexp"
	"strings"
	"unicode"
)

var nonAlphanumeric = regexp.MustCompile(`[^a-z0-9-]+`)

// Slug generates a URL-safe slug from a string
// - Lowercase
// - Replace spaces with hyphens
// - Remove non-alphanumeric characters (except hyphens)
// - Collapse multiple hyphens
// - Truncate to maxLen characters
func Slug(s string, maxLen int) string {
	// Lowercase
	s = strings.ToLower(s)

	// Replace spaces and underscores with hyphens
	s = strings.Map(func(r rune) rune {
		if r == ' ' || r == '_' {
			return '-'
		}
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' {
			return r
		}
		return -1 // Remove
	}, s)

	// Collapse multiple hyphens
	for strings.Contains(s, "--") {
		s = strings.ReplaceAll(s, "--", "-")
	}

	// Trim leading/trailing hyphens
	s = strings.Trim(s, "-")

	// Truncate
	if len(s) > maxLen {
		s = s[:maxLen]
		// Don't end with hyphen after truncation
		s = strings.TrimRight(s, "-")
	}

	return s
}
```

Update ticket.go's Slug method to use this:
```go
func (t *Ticket) Slug() string {
	return Slug(t.Title, 20)
}
```

### 4. internal/ticket/ticket_test.go

Unit tests for ticket operations:

```go
package ticket

import (
	"testing"
	"time"
)

func TestNewTicket(t *testing.T) {
	ticket := NewTicket("Test Title", "Test body content")

	if ticket.ID == "" {
		t.Error("expected ID to be set")
	}
	if ticket.Title != "Test Title" {
		t.Errorf("expected title 'Test Title', got %q", ticket.Title)
	}
	if ticket.Body != "Test body content" {
		t.Errorf("expected body 'Test body content', got %q", ticket.Body)
	}
	if ticket.Dates.Created.IsZero() {
		t.Error("expected Created to be set")
	}
	if len(ticket.Sessions) != 0 {
		t.Error("expected empty sessions")
	}
}

func TestTicketSlug(t *testing.T) {
	tests := []struct {
		title string
		want  string
	}{
		{"Add login functionality", "add-login-functional"},
		{"Fix Bug #123", "fix-bug-123"},
		{"UPPERCASE TITLE", "uppercase-title"},
		{"a  b  c", "a-b-c"},
		{"special!@#chars", "specialchars"},
		{"very-long-title-that-exceeds-twenty-characters", "very-long-title-that"},
	}

	for _, tt := range tests {
		ticket := NewTicket(tt.title, "")
		if got := ticket.Slug(); got != tt.want {
			t.Errorf("Slug(%q) = %q, want %q", tt.title, got, tt.want)
		}
	}
}

func TestAddSession(t *testing.T) {
	ticket := NewTicket("Test", "Body")
	oldUpdated := ticket.Dates.Updated

	time.Sleep(time.Millisecond) // Ensure time difference
	sess := ticket.AddSession("claude")

	if sess.ID == "" {
		t.Error("expected session ID to be set")
	}
	if sess.Agent != "claude" {
		t.Errorf("expected agent 'claude', got %q", sess.Agent)
	}
	if sess.TmuxWindow != "test" {
		t.Errorf("expected tmux_window 'test', got %q", sess.TmuxWindow)
	}
	if len(sess.StatusHistory) != 1 {
		t.Error("expected one status history entry")
	}
	if sess.StatusHistory[0].Status != "starting" {
		t.Errorf("expected status 'starting', got %q", sess.StatusHistory[0].Status)
	}
	if !ticket.Dates.Updated.After(oldUpdated) {
		t.Error("expected Updated to be updated")
	}
}

func TestActiveSession(t *testing.T) {
	ticket := NewTicket("Test", "Body")

	// No sessions
	if ticket.ActiveSession() != nil {
		t.Error("expected no active session")
	}

	// Add session
	sess := ticket.AddSession("claude")
	active := ticket.ActiveSession()
	if active == nil {
		t.Error("expected active session")
	}
	if active.ID != sess.ID {
		t.Error("expected same session")
	}

	// End session
	now := time.Now()
	sess.EndedAt = &now
	if ticket.ActiveSession() != nil {
		t.Error("expected no active session after ending")
	}
}
```

### 5. internal/ticket/store_test.go

Integration tests for store operations:

```go
package ticket

import (
	"os"
	"path/filepath"
	"testing"
)

func setupTestStore(t *testing.T) (*Store, func()) {
	t.Helper()

	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "ticket-test-*")
	if err != nil {
		t.Fatal(err)
	}

	// Create .cortex/tickets structure
	ticketsDir := filepath.Join(tmpDir, ".cortex", "tickets")
	for _, status := range []string{"backlog", "progress", "done"} {
		if err := os.MkdirAll(filepath.Join(ticketsDir, status), 0755); err != nil {
			t.Fatal(err)
		}
	}

	store := NewStore(tmpDir)
	cleanup := func() {
		os.RemoveAll(tmpDir)
	}

	return store, cleanup
}

func TestStoreCreate(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ticket := NewTicket("Test Ticket", "Test body")
	if err := store.Create(ticket); err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	// Verify file exists
	path := store.ticketPath(StatusBacklog, ticket.ID)
	if _, err := os.Stat(path); err != nil {
		t.Errorf("ticket file not created: %v", err)
	}
}

func TestStoreRead(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ticket := NewTicket("Test Ticket", "Test body")
	if err := store.Create(ticket); err != nil {
		t.Fatal(err)
	}

	read, status, err := store.Read(ticket.ID)
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if read.ID != ticket.ID {
		t.Errorf("expected ID %q, got %q", ticket.ID, read.ID)
	}
	if status != StatusBacklog {
		t.Errorf("expected status backlog, got %v", status)
	}
}

func TestStoreUpdate(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ticket := NewTicket("Original", "Body")
	if err := store.Create(ticket); err != nil {
		t.Fatal(err)
	}

	ticket.Title = "Updated"
	if err := store.Update(ticket); err != nil {
		t.Fatalf("Update failed: %v", err)
	}

	read, _, err := store.Read(ticket.ID)
	if err != nil {
		t.Fatal(err)
	}
	if read.Title != "Updated" {
		t.Errorf("expected title 'Updated', got %q", read.Title)
	}
}

func TestStoreDelete(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ticket := NewTicket("Test", "Body")
	if err := store.Create(ticket); err != nil {
		t.Fatal(err)
	}

	if err := store.Delete(ticket.ID); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	_, _, err := store.Read(ticket.ID)
	if err == nil {
		t.Error("expected error reading deleted ticket")
	}
}

func TestStoreList(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	// Create multiple tickets
	for i := 0; i < 3; i++ {
		ticket := NewTicket("Test", "Body")
		if err := store.Create(ticket); err != nil {
			t.Fatal(err)
		}
	}

	tickets, err := store.List(StatusBacklog)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}
	if len(tickets) != 3 {
		t.Errorf("expected 3 tickets, got %d", len(tickets))
	}
}

func TestStoreMove(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ticket := NewTicket("Test", "Body")
	if err := store.Create(ticket); err != nil {
		t.Fatal(err)
	}

	// Move to progress
	if err := store.Move(ticket.ID, StatusProgress); err != nil {
		t.Fatalf("Move failed: %v", err)
	}

	_, status, err := store.Read(ticket.ID)
	if err != nil {
		t.Fatal(err)
	}
	if status != StatusProgress {
		t.Errorf("expected status progress, got %v", status)
	}

	// Verify old file removed
	oldPath := store.ticketPath(StatusBacklog, ticket.ID)
	if _, err := os.Stat(oldPath); !os.IsNotExist(err) {
		t.Error("expected old file to be removed")
	}
}

func TestStoreMoveToDonetSetsApproved(t *testing.T) {
	store, cleanup := setupTestStore(t)
	defer cleanup()

	ticket := NewTicket("Test", "Body")
	if err := store.Create(ticket); err != nil {
		t.Fatal(err)
	}

	if err := store.Move(ticket.ID, StatusDone); err != nil {
		t.Fatal(err)
	}

	read, _, err := store.Read(ticket.ID)
	if err != nil {
		t.Fatal(err)
	}
	if read.Dates.Approved == nil {
		t.Error("expected Approved to be set when moving to done")
	}
}
```

### 6. Dependencies

Add to go.mod:
```
github.com/google/uuid
```

### 7. Cleanup

Remove the placeholder file:
- Delete `internal/daemon/mcp/.gitkeep` (will be handled by MCP ticket)

Actually, keep `internal/daemon/mcp/.gitkeep` - the MCP ticket will handle it.

## Verification

```bash
# Build succeeds
make build

# Tests pass
make test
# Should show ticket and store tests passing

# Lint passes
make lint
```

## Notes

- Tickets are stored as `{id}.json` in status folders
- ID is a UUID v4
- Slug is derived from title, max 20 chars
- Moving to done automatically sets `dates.approved`
- Store searches all status folders when reading by ID
- No daemon integration yet - that comes with MCP/API tickets
