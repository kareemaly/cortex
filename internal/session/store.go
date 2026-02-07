package session

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/kareemaly/cortex/internal/storage"
)

// Store manages session state backed by a single JSON file.
// Sessions are keyed by ticket short ID.
type Store struct {
	path string
	mu   sync.Mutex
}

// NewStore creates a new session store backed by the given file path.
func NewStore(path string) *Store {
	return &Store{path: path}
}

// Create adds a new session for the given ticket.
// Returns the key (ticket short ID) and the created session.
func (s *Store) Create(ticketID, agent, tmuxWindow string, worktreePath, featureBranch *string) (string, *Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sessions, err := s.load()
	if err != nil {
		return "", nil, err
	}

	key := storage.ShortID(ticketID)
	session := &Session{
		TicketID:      ticketID,
		Agent:         agent,
		TmuxWindow:    tmuxWindow,
		WorktreePath:  worktreePath,
		FeatureBranch: featureBranch,
		StartedAt:     time.Now().UTC(),
		Status:        AgentStatusStarting,
	}

	sessions[key] = session

	if err := s.save(sessions); err != nil {
		return "", nil, err
	}

	return key, session, nil
}

// Get retrieves a session by ticket short ID.
func (s *Store) Get(ticketShortID string) (*Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sessions, err := s.load()
	if err != nil {
		return nil, err
	}

	session, ok := sessions[ticketShortID]
	if !ok {
		return nil, &storage.NotFoundError{Resource: "session", ID: ticketShortID}
	}

	return session, nil
}

// GetByTicketID retrieves a session by full ticket ID.
func (s *Store) GetByTicketID(ticketID string) (*Session, error) {
	return s.Get(storage.ShortID(ticketID))
}

// UpdateStatus updates the status and optional tool for a session.
func (s *Store) UpdateStatus(ticketShortID string, status AgentStatus, tool *string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	sessions, err := s.load()
	if err != nil {
		return err
	}

	session, ok := sessions[ticketShortID]
	if !ok {
		return &storage.NotFoundError{Resource: "session", ID: ticketShortID}
	}

	session.Status = status
	session.Tool = tool

	return s.save(sessions)
}

// End removes a session entry (ephemeral — deleted on end).
func (s *Store) End(ticketShortID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	sessions, err := s.load()
	if err != nil {
		return err
	}

	if _, ok := sessions[ticketShortID]; !ok {
		return &storage.NotFoundError{Resource: "session", ID: ticketShortID}
	}

	delete(sessions, ticketShortID)

	return s.save(sessions)
}

// List returns all active sessions.
func (s *Store) List() (map[string]*Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	return s.load()
}

// load reads sessions from the JSON file. Returns empty map if file doesn't exist or is empty.
func (s *Store) load() (map[string]*Session, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]*Session), nil
		}
		return nil, fmt.Errorf("read sessions file: %w", err)
	}

	if len(data) == 0 {
		return make(map[string]*Session), nil
	}

	var sessions map[string]*Session
	if err := json.Unmarshal(data, &sessions); err != nil {
		return nil, fmt.Errorf("unmarshal sessions: %w", err)
	}

	if sessions == nil {
		sessions = make(map[string]*Session)
	}

	return sessions, nil
}

// save writes sessions to the JSON file atomically.
func (s *Store) save(sessions map[string]*Session) error {
	data, err := json.MarshalIndent(sessions, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal sessions: %w", err)
	}

	return storage.AtomicWriteFile(s.path, data)
}
