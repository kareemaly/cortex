package api

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"github.com/kareemaly/cortex1/internal/ticket"
)

// StoreManager manages per-project ticket stores.
type StoreManager struct {
	mu     sync.RWMutex
	stores map[string]*ticket.Store
	logger *slog.Logger
}

// NewStoreManager creates a new StoreManager.
func NewStoreManager(logger *slog.Logger) *StoreManager {
	return &StoreManager{
		stores: make(map[string]*ticket.Store),
		logger: logger,
	}
}

// GetStore returns the ticket store for the given project path.
// Creates a new store if one doesn't exist for the path.
func (m *StoreManager) GetStore(projectPath string) (*ticket.Store, error) {
	projectPath = filepath.Clean(projectPath)

	// Fast path: check if store already exists
	m.mu.RLock()
	store, exists := m.stores[projectPath]
	m.mu.RUnlock()

	if exists {
		return store, nil
	}

	// Slow path: create new store with write lock
	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check after acquiring write lock
	if store, exists := m.stores[projectPath]; exists {
		return store, nil
	}

	// Verify project path exists
	if _, err := os.Stat(projectPath); err != nil {
		return nil, fmt.Errorf("project path not found: %w", err)
	}

	// Create store at {projectPath}/.cortex/tickets/
	ticketsDir := filepath.Join(projectPath, ".cortex", "tickets")
	store, err := ticket.NewStore(ticketsDir)
	if err != nil {
		return nil, fmt.Errorf("failed to create ticket store: %w", err)
	}

	m.stores[projectPath] = store
	m.logger.Debug("created ticket store", "project", projectPath)

	return store, nil
}
