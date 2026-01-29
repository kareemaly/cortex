package prompt

import (
	"os"
)

// PromptResolver resolves prompt files with fallback to a base config directory.
type PromptResolver struct {
	ProjectRoot string
	BaseRoot    string // empty if no extension
}

// NewPromptResolver creates a resolver for the given project and optional base root.
func NewPromptResolver(projectRoot, baseRoot string) *PromptResolver {
	return &PromptResolver{
		ProjectRoot: projectRoot,
		BaseRoot:    baseRoot,
	}
}

// ResolveArchitectPrompt finds and loads an architect prompt file.
// Checks project first, then falls back to base if configured.
func (r *PromptResolver) ResolveArchitectPrompt(stage string) (string, error) {
	// Try project first
	projectPath := ArchitectPromptPath(r.ProjectRoot, stage)
	content, err := r.loadIfExists(projectPath)
	if err != nil {
		return "", err
	}
	if content != "" {
		return content, nil
	}

	// Try base fallback
	if r.BaseRoot != "" {
		basePath := ArchitectPromptPath(r.BaseRoot, stage)
		content, err = r.loadIfExists(basePath)
		if err != nil {
			return "", err
		}
		if content != "" {
			return content, nil
		}
	}

	// Not found in either location
	return "", &NotFoundError{Path: projectPath}
}

// ResolveTicketPrompt finds and loads a ticket prompt file.
// Checks project first, then falls back to base if configured.
func (r *PromptResolver) ResolveTicketPrompt(ticketType, stage string) (string, error) {
	// Try project first
	projectPath := TicketPromptPath(r.ProjectRoot, ticketType, stage)
	content, err := r.loadIfExists(projectPath)
	if err != nil {
		return "", err
	}
	if content != "" {
		return content, nil
	}

	// Try base fallback
	if r.BaseRoot != "" {
		basePath := TicketPromptPath(r.BaseRoot, ticketType, stage)
		content, err = r.loadIfExists(basePath)
		if err != nil {
			return "", err
		}
		if content != "" {
			return content, nil
		}
	}

	// Not found in either location
	return "", &NotFoundError{Path: projectPath}
}

// loadIfExists loads a file if it exists, returns empty string if not found.
// Returns error only for read errors (not missing files).
func (r *PromptResolver) loadIfExists(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return string(content), nil
}
