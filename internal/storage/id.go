package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func formatTimestamp(t time.Time) string {
	return t.Format("2006-01-02-1504")
}

func NewTicketID(collisionChecker func(dir string) bool, t time.Time, title string) (string, error) {
	slug := GenerateSlug(title, "ticket")
	base := formatTimestamp(t) + "-" + slug
	return resolveCollision(base, collisionChecker)
}

func NewCollabID(collisionChecker func(dir string) bool, t time.Time, slug string) (string, error) {
	if slug == "" {
		return "", fmt.Errorf("slug cannot be empty")
	}
	cleanSlug := GenerateSlug(slug, "collab")
	base := formatTimestamp(t) + "-" + cleanSlug
	return resolveCollision(base, collisionChecker)
}

func NewArchitectSessionID(collisionChecker func(dir string) bool, t time.Time) (string, error) {
	base := formatTimestamp(t)
	return resolveCollision(base, collisionChecker)
}

func MakeDirCollisionChecker(parentDir string) func(string) bool {
	return func(folderName string) bool {
		_, err := os.Stat(filepath.Join(parentDir, folderName))
		return err == nil
	}
}

func resolveCollision(base string, exists func(string) bool) (string, error) {
	if !exists(base) {
		return base, nil
	}
	for i := 2; i < 10000; i++ {
		candidate := fmt.Sprintf("%s-%d", base, i)
		if !exists(candidate) {
			return candidate, nil
		}
	}
	return "", fmt.Errorf("unable to resolve collision after 10000 attempts for %q", base)
}

func NewTicketIDFromCreated(created time.Time, title string, rootDir string) (string, error) {
	checker := func(folder string) bool {
		for _, status := range []string{"backlog", "progress", "done"} {
			if _, err := os.Stat(filepath.Join(rootDir, status, folder)); err == nil {
				return true
			}
		}
		return false
	}
	slug := GenerateSlug(title, "ticket")
	base := formatTimestamp(created) + "-" + slug
	return resolveCollision(base, checker)
}

func NewCollabIDFromPrompt(collabsDir string, created time.Time, prompt string) (string, error) {
	slug := GenerateSlug(prompt, "collab")
	return NewCollabID(MakeDirCollisionChecker(collabsDir), created, slug)
}
