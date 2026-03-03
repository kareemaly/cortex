package api

import (
	"log/slog"
	"path/filepath"
	"sync"

	"github.com/kareemaly/cortex/internal/queue"
)

type QueueManager struct {
	mu     sync.RWMutex
	stores map[string]*queue.Store
	logger *slog.Logger
}

func NewQueueManager(logger *slog.Logger) *QueueManager {
	return &QueueManager{
		stores: make(map[string]*queue.Store),
		logger: logger,
	}
}

func (m *QueueManager) GetStore(projectPath string) *queue.Store {
	m.mu.RLock()
	store, ok := m.stores[projectPath]
	m.mu.RUnlock()

	if ok {
		return store
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	if store, ok := m.stores[projectPath]; ok {
		return store
	}

	queuePath := filepath.Join(projectPath, ".queue.json")
	store = queue.NewStore(queuePath)
	m.stores[projectPath] = store

	return store
}
