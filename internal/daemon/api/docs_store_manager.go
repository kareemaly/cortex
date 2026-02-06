package api

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"

	"github.com/kareemaly/cortex/internal/docs"
	"github.com/kareemaly/cortex/internal/events"
	projectconfig "github.com/kareemaly/cortex/internal/project/config"
)

// DocsStoreManager manages per-project docs stores.
type DocsStoreManager struct {
	mu     sync.RWMutex
	stores map[string]*docs.Store
	logger *slog.Logger
	bus    *events.Bus
}

// NewDocsStoreManager creates a new DocsStoreManager.
func NewDocsStoreManager(logger *slog.Logger, bus *events.Bus) *DocsStoreManager {
	return &DocsStoreManager{
		stores: make(map[string]*docs.Store),
		logger: logger,
		bus:    bus,
	}
}

// GetStore returns the docs store for the given project path.
// Creates a new store if one doesn't exist for the path.
func (m *DocsStoreManager) GetStore(projectPath string) (*docs.Store, error) {
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

	// Load project config to resolve docs path
	cfg, err := projectconfig.Load(projectPath)
	if err != nil {
		cfg = projectconfig.DefaultConfig()
	}

	docsDir := cfg.DocsPath(projectPath)
	store, err = docs.NewStore(docsDir, m.bus, projectPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create docs store: %w", err)
	}

	m.stores[projectPath] = store
	m.logger.Debug("created docs store", "project", projectPath, "docsDir", docsDir)

	return store, nil
}
