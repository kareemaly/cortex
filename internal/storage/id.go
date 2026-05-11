package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
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

func slugFromPrompt(prompt string) string {
	slug := strings.ToLower(prompt)
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.ReplaceAll(slug, "_", "-")
	var result strings.Builder
	for _, r := range slug {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result.WriteRune(r)
		}
	}
	slug = result.String()
	for strings.Contains(slug, "--") {
		slug = strings.ReplaceAll(slug, "--", "-")
	}
	slug = strings.Trim(slug, "-")
	if len(slug) > 40 {
		slug = slug[:40]
		lastHyphen := strings.LastIndex(slug, "-")
		if lastHyphen > 0 {
			slug = slug[:lastHyphen]
		}
	}
	if slug == "" {
		return "collab"
	}
	return slug
}
