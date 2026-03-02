package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"unicode"
)

const maxSlugLength = 20

const maxTmuxNameLength = 128

func SanitizeTmuxName(name string) string {
	if name == "" {
		return ""
	}

	var result strings.Builder
	result.Grow(len(name))

	for i, r := range name {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_':
			result.WriteRune(r)
		case r == ' ' || r == '-':
			if i == 0 {
				result.WriteRune('_')
			} else {
				result.WriteRune('-')
			}
		default:
			if i == 0 {
				result.WriteRune('_')
			} else {
				result.WriteRune('-')
			}
		}
	}

	sanitized := result.String()
	if len(sanitized) > maxTmuxNameLength {
		sanitized = sanitized[:maxTmuxNameLength]
	}

	return sanitized
}

func GenerateSlug(title, fallback string) string {
	slug := strings.ToLower(title)
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.ReplaceAll(slug, "_", "-")

	var result strings.Builder
	for _, r := range slug {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' {
			result.WriteRune(r)
		}
	}
	slug = result.String()

	for strings.Contains(slug, "--") {
		slug = strings.ReplaceAll(slug, "--", "-")
	}

	slug = strings.Trim(slug, "-")

	if len(slug) > maxSlugLength {
		slug = truncateAtWordBoundary(slug, maxSlugLength)
	}

	if slug == "" {
		return fallback
	}

	return slug
}

// ShortID returns the first 8 characters of an ID.
func ShortID(id string) string {
	if len(id) > 8 {
		return id[:8]
	}
	return id
}

// DirName returns the directory name for an entity: {slug}-{shortid}.
func DirName(title, id, fallback string) string {
	slug := GenerateSlug(title, fallback)
	return fmt.Sprintf("%s-%s", slug, ShortID(id))
}

// ExpandHome expands a leading ~/ in a path to the user's home directory.
func ExpandHome(path string) string {
	if len(path) >= 2 && path[:2] == "~/" {
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return path
		}
		return filepath.Join(homeDir, path[2:])
	}
	return path
}

// truncateAtWordBoundary truncates a slug to maxLen without cutting mid-word.
func truncateAtWordBoundary(slug string, maxLen int) string {
	if len(slug) <= maxLen {
		return slug
	}

	truncated := slug[:maxLen]
	lastHyphen := strings.LastIndex(truncated, "-")

	if lastHyphen > 0 {
		return truncated[:lastHyphen]
	}

	return truncated
}
