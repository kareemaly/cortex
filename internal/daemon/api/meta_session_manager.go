package api

import (
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"github.com/kareemaly/cortex/internal/session"
)

// MetaSessionManager manages the global meta session store.
// Unlike SessionManager (per-project), this is a singleton backed by ~/.cortex/meta-session.json.
type MetaSessionManager struct {
	mu     sync.Once
	store  *session.Store
	logger *slog.Logger
}

// NewMetaSessionManager creates a new MetaSessionManager.
func NewMetaSessionManager(logger *slog.Logger) *MetaSessionManager {
	return &MetaSessionManager{
		logger: logger,
	}
}

// GetStore returns the global meta session store.
// Creates it lazily on first call.
func (m *MetaSessionManager) GetStore() *session.Store {
	m.mu.Do(func() {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			m.logger.Error("failed to get home directory for meta session store", "error", err)
			// Fall back to /tmp
			homeDir = os.TempDir()
		}

		sessionsPath := filepath.Join(homeDir, ".cortex", "meta-session.json")
		m.store = session.NewStore(sessionsPath)
		m.logger.Debug("created meta session store", "path", sessionsPath)
	})

	return m.store
}
