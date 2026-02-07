package storage

import (
	"fmt"
	"strings"
	"unicode"
)

const maxSlugLength = 20

// GenerateSlug creates a URL-friendly slug from a title.
// The slug is lowercase, uses hyphens as separators, and is max 20 characters.
// If the result would be empty, returns fallback.
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
