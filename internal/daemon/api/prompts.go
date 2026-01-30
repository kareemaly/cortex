package api

import (
	"net/http"

	projectconfig "github.com/kareemaly/cortex/internal/project/config"
	"github.com/kareemaly/cortex/internal/prompt"
)

// PromptHandlers provides HTTP handlers for prompt operations.
type PromptHandlers struct {
	deps *Dependencies
}

// NewPromptHandlers creates a new PromptHandlers with the given dependencies.
func NewPromptHandlers(deps *Dependencies) *PromptHandlers {
	return &PromptHandlers{deps: deps}
}

// ResolvePromptResponse is the response for the resolve prompt endpoint.
type ResolvePromptResponse struct {
	Content    string `json:"content"`
	SourcePath string `json:"source_path"`
}

// Resolve handles GET /prompts/resolve - resolves a prompt file with extension fallback.
// Query parameters:
//   - role: "architect" or "ticket" (required)
//   - stage: "SYSTEM", "KICKOFF", or "APPROVE" (required)
//   - type: ticket type name (required when role=ticket)
func (h *PromptHandlers) Resolve(w http.ResponseWriter, r *http.Request) {
	projectPath := GetProjectPath(r.Context())

	// Parse query parameters
	role := r.URL.Query().Get("role")
	stage := r.URL.Query().Get("stage")
	ticketType := r.URL.Query().Get("type")

	// Validate required parameters
	if role == "" {
		writeError(w, http.StatusBadRequest, "missing_parameter", "role parameter is required")
		return
	}
	if stage == "" {
		writeError(w, http.StatusBadRequest, "missing_parameter", "stage parameter is required")
		return
	}

	// Validate role
	if role != "architect" && role != "ticket" {
		writeError(w, http.StatusBadRequest, "invalid_parameter", "role must be 'architect' or 'ticket'")
		return
	}

	// Validate stage
	switch stage {
	case prompt.StageSystem, prompt.StageKickoff, prompt.StageApprove:
		// Valid
	default:
		writeError(w, http.StatusBadRequest, "invalid_parameter", "stage must be 'SYSTEM', 'KICKOFF', or 'APPROVE'")
		return
	}

	// For ticket prompts, type is required
	if role == "ticket" && ticketType == "" {
		writeError(w, http.StatusBadRequest, "missing_parameter", "type parameter is required for ticket prompts")
		return
	}

	// Load project config to get extend path
	cfg, err := projectconfig.Load(projectPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "config_error", err.Error())
		return
	}

	// Create resolver with fallback to extended base
	resolver := prompt.NewPromptResolver(projectPath, cfg.ResolvedExtendPath())

	var resolved *prompt.ResolvedPrompt
	if role == "architect" {
		resolved, err = resolver.ResolveArchitectPromptWithPath(stage)
	} else {
		resolved, err = resolver.ResolveTicketPromptWithPath(ticketType, stage)
	}

	if err != nil {
		if _, ok := err.(*prompt.NotFoundError); ok {
			writeError(w, http.StatusNotFound, "prompt_not_found", err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, "resolve_error", err.Error())
		return
	}

	resp := ResolvePromptResponse{
		Content:    resolved.Content,
		SourcePath: resolved.SourcePath,
	}
	writeJSON(w, http.StatusOK, resp)
}
