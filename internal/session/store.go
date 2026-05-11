package session

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/kareemaly/cortex/internal/storage"
)

// Store manages session state backed by a single JSON file.
//
// The on-disk map is keyed by the canonical SessionID UUID, minted at
// creation time. Architect, ticket and collab sessions share the same
// routing key so callers can address every session uniformly via
// /agent/status?session_id=<uuid>.
type Store struct {
	path string
	mu   sync.Mutex
}

// NewStore creates a new session store backed by the given file path.
func NewStore(path string) *Store {
	return &Store{path: path}
}

// NewSessionID mints a 16-byte hex UUID for a session. A failing entropy
// source is fatal — a secure RNG is a hard requirement, not something to
// paper over.
func NewSessionID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(fmt.Errorf("session: crypto/rand.Read failed: %w", err))
	}
	return hex.EncodeToString(b[:])
}

// Create adds a new ticket session and returns the created session. The
// session's SessionID field holds the canonical UUID routing key.
func (s *Store) Create(ticketID, agent, tmuxWindow string) (*Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sessions, err := s.load()
	if err != nil {
		return nil, err
	}

	sess := &Session{
		SessionID:  NewSessionID(),
		Type:       SessionTypeTicket,
		TicketID:   ticketID,
		Agent:      agent,
		TmuxWindow: tmuxWindow,
		StartedAt:  time.Now().UTC(),
		Status:     AgentStatusStarting,
	}

	sessions[sess.SessionID] = sess

	if err := s.save(sessions); err != nil {
		return nil, err
	}

	return sess, nil
}

// CreateCollab adds a new collab session.
func (s *Store) CreateCollab(collabID, prompt, agent, tmuxWindow string) (*Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sessions, err := s.load()
	if err != nil {
		return nil, err
	}

	sess := &Session{
		SessionID:  NewSessionID(),
		Type:       SessionTypeCollab,
		CollabID:   collabID,
		Prompt:     prompt,
		Agent:      agent,
		TmuxWindow: tmuxWindow,
		StartedAt:  time.Now().UTC(),
		Status:     AgentStatusStarting,
	}

	sessions[sess.SessionID] = sess

	if err := s.save(sessions); err != nil {
		return nil, err
	}

	return sess, nil
}

// GetByCollabID retrieves a collab session by its full collab ID.
func (s *Store) GetByCollabID(collabID string) (*Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sessions, err := s.load()
	if err != nil {
		return nil, err
	}
	for _, sess := range sessions {
		if sess.Type == SessionTypeCollab && sess.CollabID == collabID {
			return sess, nil
		}
	}
	return nil, &storage.NotFoundError{Resource: "session", ID: collabID}
}

// EndCollab removes a collab session entry by its collab ID.
func (s *Store) EndCollab(collabID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	sessions, err := s.load()
	if err != nil {
		return err
	}
	for id, sess := range sessions {
		if sess.Type == SessionTypeCollab && sess.CollabID == collabID {
			delete(sessions, id)
			return s.save(sessions)
		}
	}
	return &storage.NotFoundError{Resource: "session", ID: collabID}
}

// CreateArchitect adds the architect session with the given sessionID.
// There is at most one architect session per store; any existing architect
// is replaced. sessionID must be a Hiveryn-compatible timestamp ID
// (YYYY-MM-DD-HHMM).
func (s *Store) CreateArchitect(sessionID, agent, tmuxWindow string) (*Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sessions, err := s.load()
	if err != nil {
		return nil, err
	}

	// Remove any prior architect entry.
	for id, sess := range sessions {
		if sess.Type == SessionTypeArchitect {
			delete(sessions, id)
		}
	}

	sess := &Session{
		SessionID:  sessionID,
		Type:       SessionTypeArchitect,
		Agent:      agent,
		TmuxWindow: tmuxWindow,
		StartedAt:  time.Now().UTC(),
		Status:     AgentStatusStarting,
	}

	sessions[sess.SessionID] = sess

	if err := s.save(sessions); err != nil {
		return nil, err
	}

	return sess, nil
}

// GetArchitect retrieves the architect session.
func (s *Store) GetArchitect() (*Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sessions, err := s.load()
	if err != nil {
		return nil, err
	}
	for _, sess := range sessions {
		if sess.Type == SessionTypeArchitect {
			return sess, nil
		}
	}
	return nil, &storage.NotFoundError{Resource: "session", ID: ArchitectSessionKey}
}

// EndArchitect removes the architect session entry.
func (s *Store) EndArchitect() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	sessions, err := s.load()
	if err != nil {
		return err
	}
	for id, sess := range sessions {
		if sess.Type == SessionTypeArchitect {
			delete(sessions, id)
			return s.save(sessions)
		}
	}
	return &storage.NotFoundError{Resource: "session", ID: ArchitectSessionKey}
}

// GetByTicketID retrieves a ticket session by full ticket ID.
func (s *Store) GetByTicketID(ticketID string) (*Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sessions, err := s.load()
	if err != nil {
		return nil, err
	}
	for _, sess := range sessions {
		if sess.Type == SessionTypeTicket && sess.TicketID == ticketID {
			return sess, nil
		}
	}
	return nil, &storage.NotFoundError{Resource: "session", ID: ticketID}
}

// EndByTicketID removes a ticket session identified by its ticket ID.
func (s *Store) EndByTicketID(ticketID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	sessions, err := s.load()
	if err != nil {
		return err
	}
	for id, sess := range sessions {
		if sess.Type == SessionTypeTicket && sess.TicketID == ticketID {
			delete(sessions, id)
			return s.save(sessions)
		}
	}
	return &storage.NotFoundError{Resource: "session", ID: ticketID}
}

// List returns all active sessions keyed by SessionID UUID.
func (s *Store) List() (map[string]*Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.load()
}

// GetBySessionID retrieves a session by its canonical UUID.
func (s *Store) GetBySessionID(sessionID string) (*Session, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sessions, err := s.load()
	if err != nil {
		return nil, err
	}
	if sess, ok := sessions[sessionID]; ok {
		return sess, nil
	}
	return nil, &storage.NotFoundError{Resource: "session", ID: sessionID}
}

// EndBySessionID removes a session entry by its canonical UUID.
func (s *Store) EndBySessionID(sessionID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	sessions, err := s.load()
	if err != nil {
		return err
	}
	if _, ok := sessions[sessionID]; !ok {
		return &storage.NotFoundError{Resource: "session", ID: sessionID}
	}
	delete(sessions, sessionID)
	return s.save(sessions)
}

// UpdateStatusBySessionID updates status/tool/work by session UUID.
func (s *Store) UpdateStatusBySessionID(sessionID string, status AgentStatus, tool, work *string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	sessions, err := s.load()
	if err != nil {
		return err
	}
	sess, ok := sessions[sessionID]
	if !ok {
		return &storage.NotFoundError{Resource: "session", ID: sessionID}
	}
	sess.Status = status
	sess.Tool = tool
	sess.Work = work
	return s.save(sessions)
}

// load reads sessions from the JSON file. Returns empty map if file
// doesn't exist or is empty.
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
