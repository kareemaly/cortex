package spawn

import (
	"strings"
)

// GenerateWindowName creates a tmux window name from a ticket.
func GenerateWindowName(ticketTitle string) string {
	return generateSlug(ticketTitle)
}

// generateSlug creates a URL-safe slug from a title.
// Moved inline to avoid circular dependency with ticket package.
func generateSlug(title string) string {
	slug := strings.ToLower(title)
	slug = strings.ReplaceAll(slug, " ", "-")
	// Keep only alphanumeric and hyphens
	var result strings.Builder
	for _, r := range slug {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			result.WriteRune(r)
		}
	}
	slug = result.String()
	// Collapse multiple hyphens
	for strings.Contains(slug, "--") {
		slug = strings.ReplaceAll(slug, "--", "-")
	}
	// Trim leading/trailing hyphens
	slug = strings.Trim(slug, "-")
	// Limit length
	if len(slug) > 50 {
		slug = slug[:50]
	}
	return slug
}
