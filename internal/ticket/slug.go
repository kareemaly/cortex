package ticket

import (
	"strings"
	"unicode"
)

const maxSlugLength = 20

// GenerateSlug creates a URL-friendly slug from a title.
// The slug is lowercase, uses hyphens as separators, and is max 20 characters.
// If the result would be empty, returns "ticket".
func GenerateSlug(title string) string {
	// Lowercase and replace spaces/underscores with hyphens
	slug := strings.ToLower(title)
	slug = strings.ReplaceAll(slug, " ", "-")
	slug = strings.ReplaceAll(slug, "_", "-")

	// Remove non-alphanumeric except hyphens
	var result strings.Builder
	for _, r := range slug {
		if unicode.IsLetter(r) || unicode.IsDigit(r) || r == '-' {
			result.WriteRune(r)
		}
	}
	slug = result.String()

	// Collapse multiple hyphens
	for strings.Contains(slug, "--") {
		slug = strings.ReplaceAll(slug, "--", "-")
	}

	// Trim hyphens from ends
	slug = strings.Trim(slug, "-")

	// Truncate to max length without cutting mid-word
	if len(slug) > maxSlugLength {
		slug = truncateAtWordBoundary(slug, maxSlugLength)
	}

	// Fallback if empty
	if slug == "" {
		return "ticket"
	}

	return slug
}

// truncateAtWordBoundary truncates a slug to maxLen without cutting mid-word.
// Words are separated by hyphens.
func truncateAtWordBoundary(slug string, maxLen int) string {
	if len(slug) <= maxLen {
		return slug
	}

	// Find the last hyphen before or at maxLen
	truncated := slug[:maxLen]
	lastHyphen := strings.LastIndex(truncated, "-")

	if lastHyphen > 0 {
		return truncated[:lastHyphen]
	}

	// No hyphen found, just truncate (single long word)
	return truncated
}
