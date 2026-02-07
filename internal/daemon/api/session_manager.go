package api

import (
	"log/slog"
	"path/filepath"
	"sync"

	"github.com/kareemaly/cortex/internal/session"
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

	// Session store path: {projectPath}/.cortex/sessions.json
	sessionsPath := filepath.Join(projectPath, ".cortex", "sessions.json")
	store = session.NewStore(sessionsPath)

	m.stores[projectPath] = store
	m.logger.Debug("created session store", "project", projectPath)

	return store
}
