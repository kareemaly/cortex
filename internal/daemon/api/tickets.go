package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"

	"github.com/go-chi/chi/v5"
	"github.com/kareemaly/cortex1/internal/git"
	projectconfig "github.com/kareemaly/cortex1/internal/project/config"
	"github.com/kareemaly/cortex1/internal/ticket"
)

// TicketHandlers provides HTTP handlers for ticket operations.
type TicketHandlers struct {
	deps *Dependencies
}

// NewTicketHandlers creates a new TicketHandlers with the given dependencies.
func NewTicketHandlers(deps *Dependencies) *TicketHandlers {
	return &TicketHandlers{deps: deps}
}

// ListAll handles GET /tickets - lists all tickets grouped by status.
func (h *TicketHandlers) ListAll(w http.ResponseWriter, r *http.Request) {
	all, err := h.deps.TicketStore.ListAll()
	if err != nil {
		handleTicketError(w, err)
		return
	}

	resp := ListAllTicketsResponse{
		Backlog:  toSummaryList(all[ticket.StatusBacklog], ticket.StatusBacklog),
		Progress: toSummaryList(all[ticket.StatusProgress], ticket.StatusProgress),
		Done:     toSummaryList(all[ticket.StatusDone], ticket.StatusDone),
	}

	writeJSON(w, http.StatusOK, resp)
}

// ListByStatus handles GET /tickets/{status} - lists tickets with a specific status.
func (h *TicketHandlers) ListByStatus(w http.ResponseWriter, r *http.Request) {
	status := chi.URLParam(r, "status")
	if !validStatus(status) {
		writeError(w, http.StatusBadRequest, "invalid_status", "invalid status: must be backlog, progress, or done")
		return
	}

	tickets, err := h.deps.TicketStore.List(ticket.Status(status))
	if err != nil {
		handleTicketError(w, err)
		return
	}

	resp := ListTicketsResponse{
		Tickets: toSummaryList(tickets, ticket.Status(status)),
	}

	writeJSON(w, http.StatusOK, resp)
}

// Create handles POST /tickets - creates a new ticket.
func (h *TicketHandlers) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateTicketRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid JSON in request body")
		return
	}

	t, err := h.deps.TicketStore.Create(req.Title, req.Body)
	if err != nil {
		handleTicketError(w, err)
		return
	}

	resp := toTicketResponse(t, ticket.StatusBacklog)
	writeJSON(w, http.StatusCreated, resp)
}

// Get handles GET /tickets/{status}/{id} - gets a specific ticket.
func (h *TicketHandlers) Get(w http.ResponseWriter, r *http.Request) {
	status := chi.URLParam(r, "status")
	if !validStatus(status) {
		writeError(w, http.StatusBadRequest, "invalid_status", "invalid status: must be backlog, progress, or done")
		return
	}

	id := chi.URLParam(r, "id")
	t, actualStatus, err := h.deps.TicketStore.Get(id)
	if err != nil {
		handleTicketError(w, err)
		return
	}

	// Verify the ticket is in the expected status
	if string(actualStatus) != status {
		writeError(w, http.StatusNotFound, "not_found", "ticket not found in specified status")
		return
	}

	resp := toTicketResponse(t, actualStatus)
	writeJSON(w, http.StatusOK, resp)
}

// Update handles PUT /tickets/{status}/{id} - updates a ticket.
func (h *TicketHandlers) Update(w http.ResponseWriter, r *http.Request) {
	status := chi.URLParam(r, "status")
	if !validStatus(status) {
		writeError(w, http.StatusBadRequest, "invalid_status", "invalid status: must be backlog, progress, or done")
		return
	}

	id := chi.URLParam(r, "id")

	// Check ticket exists and is in the expected status
	_, actualStatus, err := h.deps.TicketStore.Get(id)
	if err != nil {
		handleTicketError(w, err)
		return
	}
	if string(actualStatus) != status {
		writeError(w, http.StatusNotFound, "not_found", "ticket not found in specified status")
		return
	}

	var req UpdateTicketRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid JSON in request body")
		return
	}

	t, err := h.deps.TicketStore.Update(id, req.Title, req.Body)
	if err != nil {
		handleTicketError(w, err)
		return
	}

	resp := toTicketResponse(t, actualStatus)
	writeJSON(w, http.StatusOK, resp)
}

// Delete handles DELETE /tickets/{status}/{id} - deletes a ticket.
func (h *TicketHandlers) Delete(w http.ResponseWriter, r *http.Request) {
	status := chi.URLParam(r, "status")
	if !validStatus(status) {
		writeError(w, http.StatusBadRequest, "invalid_status", "invalid status: must be backlog, progress, or done")
		return
	}

	id := chi.URLParam(r, "id")

	// Check ticket exists and is in the expected status
	_, actualStatus, err := h.deps.TicketStore.Get(id)
	if err != nil {
		handleTicketError(w, err)
		return
	}
	if string(actualStatus) != status {
		writeError(w, http.StatusNotFound, "not_found", "ticket not found in specified status")
		return
	}

	if err := h.deps.TicketStore.Delete(id); err != nil {
		handleTicketError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Move handles POST /tickets/{status}/{id}/move - moves a ticket to a different status.
func (h *TicketHandlers) Move(w http.ResponseWriter, r *http.Request) {
	status := chi.URLParam(r, "status")
	if !validStatus(status) {
		writeError(w, http.StatusBadRequest, "invalid_status", "invalid status: must be backlog, progress, or done")
		return
	}

	id := chi.URLParam(r, "id")

	// Check ticket exists and is in the expected status
	_, actualStatus, err := h.deps.TicketStore.Get(id)
	if err != nil {
		handleTicketError(w, err)
		return
	}
	if string(actualStatus) != status {
		writeError(w, http.StatusNotFound, "not_found", "ticket not found in specified status")
		return
	}

	var req MoveTicketRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid JSON in request body")
		return
	}

	if !validStatus(req.To) {
		writeError(w, http.StatusBadRequest, "invalid_status", "invalid target status: must be backlog, progress, or done")
		return
	}

	if err := h.deps.TicketStore.Move(id, ticket.Status(req.To)); err != nil {
		handleTicketError(w, err)
		return
	}

	// Fetch the updated ticket
	t, newStatus, err := h.deps.TicketStore.Get(id)
	if err != nil {
		handleTicketError(w, err)
		return
	}

	resp := toTicketResponse(t, newStatus)
	writeJSON(w, http.StatusOK, resp)
}

// Spawn handles POST /tickets/{status}/{id}/spawn - spawns a session.
func (h *TicketHandlers) Spawn(w http.ResponseWriter, r *http.Request) {
	status := chi.URLParam(r, "status")
	if !validStatus(status) {
		writeError(w, http.StatusBadRequest, "invalid_status", "invalid status: must be backlog, progress, or done")
		return
	}

	id := chi.URLParam(r, "id")

	// Get ticket and verify status
	t, actualStatus, err := h.deps.TicketStore.Get(id)
	if err != nil {
		handleTicketError(w, err)
		return
	}
	if string(actualStatus) != status {
		writeError(w, http.StatusNotFound, "not_found", "ticket not found in specified status")
		return
	}

	// Check tmux is available
	if h.deps.TmuxManager == nil {
		writeError(w, http.StatusServiceUnavailable, "tmux_unavailable", "tmux is not installed")
		return
	}

	// Check no active sessions exist
	if t.HasActiveSessions() {
		writeError(w, http.StatusConflict, "session_active", "ticket already has an active session")
		return
	}

	// Capture git base for each repo in config
	gitBase := make(map[string]string)
	for _, repo := range h.deps.ProjectConfig.Git.Repos {
		repoPath := repo.Path
		if !filepath.IsAbs(repoPath) {
			repoPath = filepath.Join(h.deps.ProjectRoot, repoPath)
		}
		sha, err := git.GetCommitSHA(repoPath, false)
		if err != nil {
			h.deps.Logger.Warn("failed to capture git base", "repo", repoPath, "error", err)
			continue
		}
		gitBase[repo.Path] = sha
	}

	// Generate MCP config file
	mcpConfigPath, err := h.writeMCPConfig(id)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "mcp_config_error", "failed to write MCP config")
		return
	}

	// Build agent command
	agentCmd := h.buildAgentCommand(mcpConfigPath, t)

	// Get session name and window name
	sessionName := h.deps.ProjectConfig.Name
	if sessionName == "" {
		sessionName = "cortex"
	}
	windowName := ticket.GenerateSlug(t.Title)

	// Spawn agent in tmux
	_, err = h.deps.TmuxManager.SpawnAgent(sessionName, windowName, agentCmd)
	if err != nil {
		h.deps.Logger.Error("failed to spawn agent", "error", err)
		writeError(w, http.StatusInternalServerError, "tmux_error", "failed to spawn tmux window")
		return
	}

	// Add session to ticket
	session, err := h.deps.TicketStore.AddSession(id, string(h.deps.ProjectConfig.Agent), windowName, gitBase)
	if err != nil {
		h.deps.Logger.Error("failed to add session", "error", err)
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to record session")
		return
	}

	// Move ticket to progress if in backlog
	if actualStatus == ticket.StatusBacklog {
		if err := h.deps.TicketStore.Move(id, ticket.StatusProgress); err != nil {
			h.deps.Logger.Warn("failed to move ticket to progress", "error", err)
		}
	}

	// Fetch updated ticket
	t, newStatus, err := h.deps.TicketStore.Get(id)
	if err != nil {
		handleTicketError(w, err)
		return
	}

	resp := SpawnResponse{
		Session: toSessionResponse(*session),
		Ticket:  toTicketResponse(t, newStatus),
	}
	writeJSON(w, http.StatusCreated, resp)
}

// writeMCPConfig writes the MCP configuration to a temp file.
func (h *TicketHandlers) writeMCPConfig(ticketID string) (string, error) {
	mcpConfig := map[string]any{
		"mcpServers": map[string]any{
			"cortex": map[string]any{
				"command": "cortexd",
				"args":    []string{"mcp", "--ticket-id", ticketID},
			},
		},
	}

	data, err := json.Marshal(mcpConfig)
	if err != nil {
		return "", fmt.Errorf("marshal mcp config: %w", err)
	}

	// Write to temp file
	tmpFile, err := os.CreateTemp("", "cortex-mcp-*.json")
	if err != nil {
		return "", fmt.Errorf("create temp file: %w", err)
	}

	if _, err := tmpFile.Write(data); err != nil {
		_ = tmpFile.Close()
		return "", fmt.Errorf("write temp file: %w", err)
	}

	if err := tmpFile.Close(); err != nil {
		return "", fmt.Errorf("close temp file: %w", err)
	}

	return tmpFile.Name(), nil
}

// buildAgentCommand builds the command to run the agent.
func (h *TicketHandlers) buildAgentCommand(mcpConfigPath string, t *ticket.Ticket) string {
	switch h.deps.ProjectConfig.Agent {
	case projectconfig.AgentOpenCode:
		return fmt.Sprintf("opencode --mcp-config %s", mcpConfigPath)
	case projectconfig.AgentClaude:
		fallthrough
	default:
		// Build prompt from ticket
		prompt := fmt.Sprintf("Work on the following ticket:\\n\\nTitle: %s\\n\\n%s", t.Title, t.Body)
		return fmt.Sprintf("claude --mcp-config %s -p '%s'", mcpConfigPath, prompt)
	}
}
