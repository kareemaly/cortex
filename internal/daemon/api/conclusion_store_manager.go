package api

import (
	"fmt"
	"log/slog"
	"path/filepath"
	"sync"

	"github.com/kareemaly/cortex/internal/conclusion"
	"github.com/kareemaly/cortex/internal/events"
)

// ConclusionStoreManager manages per-project conclusion stores.
type ConclusionStoreManager struct {
	stores map[string]*conclusion.Store
	mu     sync.RWMutex
	logger *slog.Logger
	bus    *events.Bus
}

// NewConclusionStoreManager creates a new ConclusionStoreManager.
func NewConclusionStoreManager(logger *slog.Logger, bus *events.Bus) *ConclusionStoreManager {
	return &ConclusionStoreManager{
		stores: make(map[string]*conclusion.Store),
		logger: logger,
		bus:    bus,
	}
}

// GetStore returns (or creates) the conclusion store for a project.
func (m *ConclusionStoreManager) GetStore(projectPath string) (*conclusion.Store, error) {
	m.mu.RLock()
	store, ok := m.stores[projectPath]
	m.mu.RUnlock()
	if ok {
		return store, nil
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check after acquiring write lock
	if store, ok := m.stores[projectPath]; ok {
		return store, nil
	}

	sessionsDir := filepath.Join(projectPath, "sessions")
	store, err := conclusion.NewStore(sessionsDir, m.bus, projectPath)
	if err != nil {
		return nil, fmt.Errorf("create conclusion store for %s: %w", projectPath, err)
	}

	m.stores[projectPath] = store
	return store, nil
}
