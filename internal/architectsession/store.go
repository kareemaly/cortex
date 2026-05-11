package architectsession

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/kareemaly/cortex/internal/storage"
)

const conclusionFileName = "conclusion.md"

type ConclusionMeta struct {
	StartedAt   time.Time `yaml:"started_at"`
	ConcludedAt time.Time `yaml:"concluded_at"`
	Agent       string    `yaml:"agent"`
	Profile     string    `yaml:"profile,omitempty"`
}

type Conclusion struct {
	ID   string
	Meta ConclusionMeta
	Body string
}

func Dir(projectPath string) string {
	return filepath.Join(projectPath, "architect-sessions")
}

func EnsureDir(projectPath string) error {
	return os.MkdirAll(Dir(projectPath), 0755)
}

func WriteConclusion(projectPath, sessionID string, meta ConclusionMeta, body string) error {
	dir := filepath.Join(Dir(projectPath), sessionID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create architect session dir: %w", err)
	}

	data, err := storage.SerializeFrontmatter(&meta, body)
	if err != nil {
		return fmt.Errorf("serialize conclusion: %w", err)
	}

	return storage.AtomicWriteFile(filepath.Join(dir, conclusionFileName), data)
}

func ReadConclusion(projectPath, sessionID string) (*Conclusion, error) {
	path := filepath.Join(Dir(projectPath), sessionID, conclusionFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	meta, body, err := storage.ParseFrontmatter[ConclusionMeta](data)
	if err != nil {
		return nil, err
	}

	return &Conclusion{
		ID:   sessionID,
		Meta: *meta,
		Body: body,
	}, nil
}

func List(projectPath string) ([]*Conclusion, error) {
	dir := Dir(projectPath)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var conclusions []*Conclusion
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name()[0] == '.' {
			continue
		}
		c, err := ReadConclusion(projectPath, entry.Name())
		if err != nil {
			continue
		}
		conclusions = append(conclusions, c)
	}

	return conclusions, nil
}
