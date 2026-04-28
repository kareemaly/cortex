package api

import (
	"log/slog"
	"path/filepath"
	"sync"

	"github.com/kareemaly/cortex/internal/session"
	"github.com/kareemaly/cortex/internal/storage"
)

// SessionManager manages per-project session stores.
type SessionManager struct {
	mu     sync.RWMutex
	stores map[string]*session.Store
	logger *slog.Logger
}

// NewSessionManager creates a new SessionManager.
func NewSessionManager(logger *slog.Logger) *SessionManager {
	return &SessionManager{
		stores: make(map[string]*session.Store),
		logger: logger,
	}
}

// TotalSessionCount returns the number of active sessions across every
// known architect. Used by /agent/status/debug to verify supervisors are
// actually running.
func (m *SessionManager) TotalSessionCount() int {
	m.mu.RLock()
	stores := make([]*session.Store, 0, len(m.stores))
	for _, s := range m.stores {
		stores = append(stores, s)
	}
	m.mu.RUnlock()

	total := 0
	for _, s := range stores {
		if sessions, err := s.List(); err == nil {
			total += len(sessions)
		}
	}
	return total
}

// SetAgentSessionIDBySessionID finds the session with the given cortex UUID
// across all known architect session stores and records agentSessionID as its
// AgentSessionID. Returns nil on success, storage.NotFoundError when no
// matching session exists.
func (m *SessionManager) SetAgentSessionIDBySessionID(cortexSessionID, agentSessionID string) error {
	m.mu.RLock()
	stores := make([]*session.Store, 0, len(m.stores))
	for _, s := range m.stores {
		stores = append(stores, s)
	}
	m.mu.RUnlock()

	for _, store := range stores {
		err := store.SetAgentSessionID(cortexSessionID, agentSessionID)
		if err == nil {
			return nil
		}
		if !storage.IsNotFound(err) {
			return err
		}
	}
	return &storage.NotFoundError{Resource: "session", ID: cortexSessionID}
}

// GetStore returns the session store for the given project path.
// Creates a new store if one doesn't exist for the path.
func (m *SessionManager) GetStore(projectPath string) *session.Store {
	projectPath = filepath.Clean(projectPath)

	// Fast path: check if store already exists
	m.mu.RLock()
	store, exists := m.stores[projectPath]
	m.mu.RUnlock()

	if exists {
		return store
	}

	// Slow path: create new store with write lock
	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check after acquiring write lock
	if store, exists := m.stores[projectPath]; exists {
		return store
	}

	// Session store path: {projectPath}/.sessions.json (hidden file at root)
	sessionsPath := filepath.Join(projectPath, ".sessions.json")
	store = session.NewStore(sessionsPath)

	m.stores[projectPath] = store
	m.logger.Debug("created session store", "project", projectPath)

	return store
}
