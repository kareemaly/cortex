package entity

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/kareemaly/cortex/internal/events"
	"github.com/kareemaly/cortex/internal/storage"
)

type BaseStore struct {
	rootDir     string
	bus         *events.Bus
	projectPath string
}

func NewBaseStore(rootDir string, bus *events.Bus, projectPath string) (*BaseStore, error) {
	if err := os.MkdirAll(rootDir, 0755); err != nil {
		return nil, fmt.Errorf("create root directory %s: %w", rootDir, err)
	}

	return &BaseStore{
		rootDir:     rootDir,
		bus:         bus,
		projectPath: projectPath,
	}, nil
}

func (s *BaseStore) RootDir() string {
	return s.rootDir
}

func (s *BaseStore) FindEntityDir(resource, id string, subdirs ...string) (string, error) {
	searchDirs := []string{s.rootDir}
	if len(subdirs) > 0 {
		for _, subdir := range subdirs {
			searchDirs = append(searchDirs, filepath.Join(s.rootDir, subdir))
		}
	}

	shortID := storage.ShortID(id)

	for _, dir := range searchDirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return "", fmt.Errorf("read directory: %w", err)
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			name := entry.Name()
			if strings.HasSuffix(name, "-"+shortID) || strings.HasSuffix(name, "-"+id) {
				return filepath.Join(dir, name), nil
			}
		}
	}

	return "", &storage.NotFoundError{Resource: resource, ID: id}
}

func (s *BaseStore) LoadIndexBytes(entityDir string) ([]byte, error) {
	return os.ReadFile(filepath.Join(entityDir, "index.md"))
}

func (s *BaseStore) WriteIndexBytes(entityDir string, data []byte) error {
	target := filepath.Join(entityDir, "index.md")
	return storage.AtomicWriteFile(target, data)
}

func (s *BaseStore) ListEntries(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("read directory: %w", err)
	}

	var entityDirs []string
	for _, entry := range entries {
		if !entry.IsDir() || strings.HasPrefix(entry.Name(), ".") {
			continue
		}
		entityDirs = append(entityDirs, filepath.Join(dir, entry.Name()))
	}

	return entityDirs, nil
}

func (s *BaseStore) Emit(eventType events.EventType, ticketID string, payload any) {
	if s.bus == nil {
		return
	}
	s.bus.Emit(events.Event{
		Type:          eventType,
		ArchitectPath: s.projectPath,
		TicketID:      ticketID,
		Payload:       payload,
	})
}
