package prompt

import (
	"fmt"
	"strings"
)

// NotFoundError indicates a prompt file was not found.
type NotFoundError struct {
	Role        string   // "architect" or ticket type (work, research)
	TicketType  string   // for ticket prompts only (deprecated, kept for compatibility)
	Stage       string   // SYSTEM, KICKOFF
	SearchPaths []string // all paths checked
}

func (e *NotFoundError) Error() string {
	var sb strings.Builder
	if e.Role != "architect" {
		sb.WriteString(fmt.Sprintf("prompt not found for %s/%s", e.Role, e.Stage))
	} else {
		sb.WriteString(fmt.Sprintf("prompt not found for %s/%s", e.Role, e.Stage))
	}
	if len(e.SearchPaths) > 0 {
		sb.WriteString(", searched:\n")
		for _, p := range e.SearchPaths {
			sb.WriteString(fmt.Sprintf("  - %s\n", p))
		}
	}
	sb.WriteString("Run 'cortex init' to create default prompts")
	return sb.String()
}
