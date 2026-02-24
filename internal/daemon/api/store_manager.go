package api

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"github.com/kareemaly/cortex/internal/events"
	architectconfig "github.com/kareemaly/cortex/internal/architect/config"
	"github.com/kareemaly/cortex/internal/ticket"
)

// StoreManager manages per-project ticket stores.
type StoreManager struct {
	mu     sync.RWMutex
	stores map[string]*ticket.Store
	logger *slog.Logger
	bus    *events.Bus
}

// NewStoreManager creates a new StoreManager.
func NewStoreManager(logger *slog.Logger, bus *events.Bus) *StoreManager {
	return &StoreManager{
		stores: make(map[string]*ticket.Store),
		logger: logger,
		bus:    bus,
	}
}

// GetStore returns the ticket store for the given project path.
// Creates a new store if one doesn't exist for the path.
func (m *StoreManager) GetStore(architectPath string) (*ticket.Store, error) {
	architectPath = filepath.Clean(architectPath)

	// Fast path: check if store already exists
	m.mu.RLock()
	store, exists := m.stores[architectPath]
	m.mu.RUnlock()

	if exists {
		return store, nil
	}

	// Slow path: create new store with write lock
	m.mu.Lock()
	defer m.mu.Unlock()

	// Double-check after acquiring write lock
	if store, exists := m.stores[architectPath]; exists {
		return store, nil
	}

	// Verify project path exists
	if _, err := os.Stat(architectPath); err != nil {
		return nil, fmt.Errorf("project path not found: %w", err)
	}

	// Load project config to resolve tickets path
	cfg, err := architectconfig.Load(architectPath)
	if err != nil {
		cfg = architectconfig.DefaultConfig()
	}

	ticketsDir := cfg.TicketsPath(architectPath)
	store, err = ticket.NewStore(ticketsDir, m.bus, architectPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create ticket store: %w", err)
	}

	m.stores[architectPath] = store
	m.logger.Debug("created ticket store", "project", architectPath)

	return store, nil
}
