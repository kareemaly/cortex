package collab

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/kareemaly/cortex/internal/storage"
)

const promptFileName = "prompt.md"
const conclusionFileName = "conclusion.md"

type PromptMeta struct {
	Created time.Time `yaml:"created"`
	Agent   string    `yaml:"agent"`
	Profile string    `yaml:"profile,omitempty"`
}

type ConclusionMeta struct {
	StartedAt   time.Time `yaml:"started_at"`
	ConcludedAt time.Time `yaml:"concluded_at"`
	Agent       string    `yaml:"agent"`
	Profile     string    `yaml:"profile,omitempty"`
}

type Collab struct {
	ID             string
	PromptMeta     PromptMeta
	PromptBody     string
	ConclusionMeta *ConclusionMeta
	ConclusionBody string
}

func Dir(projectPath string) string {
	return filepath.Join(projectPath, "collabs")
}

func EnsureDir(projectPath string) error {
	return os.MkdirAll(Dir(projectPath), 0755)
}

func NewID(collabsDir string, created time.Time, slug string) (string, error) {
	cleanSlug := storage.GenerateSlug(slug, "collab")
	checker := storage.MakeDirCollisionChecker(collabsDir)
	return storage.NewCollabID(checker, created, cleanSlug)
}

func WritePrompt(projectPath, collabID string, meta PromptMeta, body string) error {
	dir := filepath.Join(Dir(projectPath), collabID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create collab dir: %w", err)
	}

	data, err := storage.SerializeFrontmatter(&meta, body)
	if err != nil {
		return fmt.Errorf("serialize prompt: %w", err)
	}

	return storage.AtomicWriteFile(filepath.Join(dir, promptFileName), data)
}

func WriteConclusion(projectPath, collabID string, meta ConclusionMeta, body string) error {
	dir := filepath.Join(Dir(projectPath), collabID)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create collab dir: %w", err)
	}

	data, err := storage.SerializeFrontmatter(&meta, body)
	if err != nil {
		return fmt.Errorf("serialize conclusion: %w", err)
	}

	return storage.AtomicWriteFile(filepath.Join(dir, conclusionFileName), data)
}

func ReadPrompt(projectPath, collabID string) (*Collab, error) {
	path := filepath.Join(Dir(projectPath), collabID, promptFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	meta, body, err := storage.ParseFrontmatter[PromptMeta](data)
	if err != nil {
		return nil, err
	}

	c := &Collab{
		ID:         collabID,
		PromptMeta: *meta,
		PromptBody: body,
	}

	if concPath := filepath.Join(Dir(projectPath), collabID, conclusionFileName); true {
		if concData, concErr := os.ReadFile(concPath); concErr == nil {
			concMeta, concBody, parseErr := storage.ParseFrontmatter[ConclusionMeta](concData)
			if parseErr == nil {
				c.ConclusionMeta = concMeta
				c.ConclusionBody = concBody
			}
		}
	}

	return c, nil
}

func ReadConclusion(projectPath, collabID string) (*ConclusionMeta, string, error) {
	path := filepath.Join(Dir(projectPath), collabID, conclusionFileName)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", err
	}

	return storage.ParseFrontmatter[ConclusionMeta](data)
}

func List(projectPath string) ([]*Collab, error) {
	dir := Dir(projectPath)
	entries, err := os.ReadDir(dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}

	var collabs []*Collab
	for _, entry := range entries {
		if !entry.IsDir() || entry.Name()[0] == '.' {
			continue
		}
		c, err := ReadPrompt(projectPath, entry.Name())
		if err != nil {
			continue
		}
		collabs = append(collabs, c)
	}

	return collabs, nil
}
