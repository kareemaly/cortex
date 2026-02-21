package api

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"github.com/kareemaly/cortex/internal/events"
	"github.com/kareemaly/cortex/internal/notes"
)

// NotesStoreManager manages per-project notes stores.
type NotesStoreManager struct {
	mu     sync.RWMutex
	stores map[string]*notes.Store
	logger *slog.Logger
	bus    *events.Bus
}

// NewNotesStoreManager creates a new NotesStoreManager.
func NewNotesStoreManager(logger *slog.Logger, bus *events.Bus) *NotesStoreManager {
	return &NotesStoreManager{
		stores: make(map[string]*notes.Store),
		logger: logger,
		bus:    bus,
	}
}

// GetStore returns the notes store for the given project path.
// Creates a new store if one doesn't exist for the path.
func (m *NotesStoreManager) GetStore(projectPath string) (*notes.Store, error) {
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

	notesPath := filepath.Join(projectPath, "notes.yaml")
	store = notes.NewStore(notesPath, m.bus, projectPath)

	m.stores[projectPath] = store
	m.logger.Debug("created notes store", "project", projectPath, "notesPath", notesPath)

	return store, nil
}
