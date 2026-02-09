package prompt

import (
	"os"
)

// PromptResolver resolves prompt files with fallback to a base config directory.
type PromptResolver struct {
	ProjectRoot string
	BaseRoot    string // empty if no extension
}

// ResolvedPrompt contains the prompt content and its source path.
type ResolvedPrompt struct {
	Content    string
	SourcePath string
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
	resolved, err := r.ResolveArchitectPromptWithPath(stage)
	if err != nil {
		return "", err
	}
	return resolved.Content, nil
}

// ResolveArchitectPromptWithPath finds and loads an architect prompt file,
// returning both content and source path. Checks project first, then falls back to base.
func (r *PromptResolver) ResolveArchitectPromptWithPath(stage string) (*ResolvedPrompt, error) {
	var searchPaths []string

	// Try project first
	projectPath := ArchitectPromptPath(r.ProjectRoot, stage)
	searchPaths = append(searchPaths, projectPath)
	content, err := r.loadIfExists(projectPath)
	if err != nil {
		return nil, err
	}
	if content != "" {
		return &ResolvedPrompt{Content: content, SourcePath: projectPath}, nil
	}

	// Try base fallback
	if r.BaseRoot != "" {
		basePath := BaseArchitectPromptPath(r.BaseRoot, stage)
		searchPaths = append(searchPaths, basePath)
		content, err = r.loadIfExists(basePath)
		if err != nil {
			return nil, err
		}
		if content != "" {
			return &ResolvedPrompt{Content: content, SourcePath: basePath}, nil
		}
	}

	// Not found in either location
	return nil, &NotFoundError{
		Role:        "architect",
		Stage:       stage,
		SearchPaths: searchPaths,
	}
}

// ResolveTicketPrompt finds and loads a ticket prompt file.
// Checks project first, then falls back to base if configured.
func (r *PromptResolver) ResolveTicketPrompt(ticketType, stage string) (string, error) {
	resolved, err := r.ResolveTicketPromptWithPath(ticketType, stage)
	if err != nil {
		return "", err
	}
	return resolved.Content, nil
}

// ResolveTicketPromptWithPath finds and loads a ticket prompt file,
// returning both content and source path. Checks project first, then falls back to base.
func (r *PromptResolver) ResolveTicketPromptWithPath(ticketType, stage string) (*ResolvedPrompt, error) {
	var searchPaths []string

	// Try project first
	projectPath := TicketPromptPath(r.ProjectRoot, ticketType, stage)
	searchPaths = append(searchPaths, projectPath)
	content, err := r.loadIfExists(projectPath)
	if err != nil {
		return nil, err
	}
	if content != "" {
		return &ResolvedPrompt{Content: content, SourcePath: projectPath}, nil
	}

	// Try base fallback
	if r.BaseRoot != "" {
		basePath := BaseTicketPromptPath(r.BaseRoot, ticketType, stage)
		searchPaths = append(searchPaths, basePath)
		content, err = r.loadIfExists(basePath)
		if err != nil {
			return nil, err
		}
		if content != "" {
			return &ResolvedPrompt{Content: content, SourcePath: basePath}, nil
		}
	}

	// Not found in either location
	return nil, &NotFoundError{
		Role:        "ticket",
		TicketType:  ticketType,
		Stage:       stage,
		SearchPaths: searchPaths,
	}
}

// ResolveMetaPrompt finds and loads a meta prompt file.
// Checks base config path only (meta prompts are global, not per-project).
func (r *PromptResolver) ResolveMetaPrompt(stage string) (string, error) {
	resolved, err := r.ResolveMetaPromptWithPath(stage)
	if err != nil {
		return "", err
	}
	return resolved.Content, nil
}

// ResolveMetaPromptWithPath finds and loads a meta prompt file,
// returning both content and source path. Checks base config path.
func (r *PromptResolver) ResolveMetaPromptWithPath(stage string) (*ResolvedPrompt, error) {
	var searchPaths []string

	// Try base fallback (meta prompts live in defaults, not per-project)
	if r.BaseRoot != "" {
		basePath := BaseMetaPromptPath(r.BaseRoot, stage)
		searchPaths = append(searchPaths, basePath)
		content, err := r.loadIfExists(basePath)
		if err != nil {
			return nil, err
		}
		if content != "" {
			return &ResolvedPrompt{Content: content, SourcePath: basePath}, nil
		}
	}

	// Not found
	return nil, &NotFoundError{
		Role:        "meta",
		Stage:       stage,
		SearchPaths: searchPaths,
	}
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
