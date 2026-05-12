package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/kareemaly/cortex/internal/cli/sdk"
	"github.com/kareemaly/cortex/internal/daemon/api"
	"github.com/kareemaly/cortex/internal/ticket"
	"github.com/kareemaly/cortex/internal/types"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// registerArchitectTools registers all tools available to architect sessions.
func (s *Server) registerArchitectTools() {
	// List tickets
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "listTickets",
		Description: "List tickets by status. Status parameter is required and must be one of: backlog, progress, done.",
	}, s.handleListTickets)

	// Read ticket
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "readTicket",
		Description: "Read full ticket details by ID",
	}, s.handleReadTicket)

	// Create work ticket
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "createWorkTicket",
		Description: "Create a new work ticket in backlog. Requires a repo field — provide a stable repo key from cortex.yaml.",
	}, s.handleCreateWorkTicket)

	// Update ticket
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "updateTicket",
		Description: "Update mutable ticket fields. Accepts: id (required), title, body, dueDate, references. dueDate must be RFC3339 when set, and an explicit empty string clears it. Use editTicketBody for targeted body edits; keep updateTicket for full-body rewrites. Does NOT support updating type, repo, status, or any other fields.",
	}, s.handleUpdateTicket)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "editTicketBody",
		Description: "Edit part of a ticket body using oldString/newString replacement. Preferred over updateTicket for body edits because it avoids full-body JSON serialization issues.",
	}, s.handleEditTicketBody)

	// Delete ticket
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "deleteTicket",
		Description: "Delete a ticket by ID",
	}, s.handleDeleteTicket)

	// Move ticket
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "moveTicket",
		Description: "Move a ticket to a different status",
	}, s.handleMoveTicket)

	// List variants
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "listVariants",
		Description: "List available agent variant names from the agents map in cortex.yaml. Use the returned names as the variant parameter in spawnSession and spawnCollabSession.",
	}, s.handleListVariants)

	// Spawn session
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "spawnSession",
		Description: "Spawn a new agent session for a ticket. Use listVariants to see available agent variant names, then pass the chosen name as the variant parameter.",
	}, s.handleSpawnSession)

	// List conclusions (persistent conclusion records)
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "listConclusions",
		Description: "List persistent conclusion records (metadata only, no body). Paginated, newest first. Use readConclusion to fetch full body.",
	}, s.handleListConclusions)

	// Read conclusion (persistent conclusion record)
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "readConclusion",
		Description: "Read a conclusion record by ID, including the full body.",
	}, s.handleReadConclusion)

	// Search across tickets and conclusions
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "search",
		Description: "Search across all tickets (all statuses) and conclusions by a case-insensitive substring query. Returns tickets in readTicket shape (with conclusion nested if done) and bare ticketless conclusions. Results sorted newest-updated first.",
	}, s.handleSearch)

	// Conclude architect session
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "concludeSession",
		Description: "Conclude the architect session and clean up. Include tickets touched, key user requests, decisions, blockers, and next steps.",
	}, s.handleArchitectConcludeSession)

	// Spawn collab session
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "spawnCollabSession",
		Description: "Spawn a ticketless collab session at any valid filesystem path with a kickoff prompt. The agent starts immediately in the directory with the given prompt as context.",
	}, s.handleSpawnCollabSession)
}

// handleListTickets lists tickets by status.
func (s *Server) handleListTickets(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input ListTicketsInput,
) (*mcp.CallToolResult, ListTicketsOutput, error) {
	// Validate status is provided and valid
	if input.Status == "" {
		return nil, ListTicketsOutput{}, NewValidationError("status", "is required")
	}
	if input.Status != "backlog" && input.Status != "progress" && input.Status != "done" {
		return nil, ListTicketsOutput{}, NewValidationError("status", "must be one of: backlog, progress, done")
	}

	resp, err := s.sdkClient.ListTicketsByStatus(input.Status, input.Query, nil)
	if err != nil {
		return nil, ListTicketsOutput{}, wrapSDKError(err)
	}

	// Map shared types to MCP-specific summaries
	summaries := make([]TicketSummary, len(resp.Tickets))
	for i, t := range resp.Tickets {
		summaries[i] = ticketSummaryResponseToMCP(&t)
	}

	return nil, ListTicketsOutput{
		Tickets: summaries,
		Total:   len(summaries),
	}, nil
}

// handleReadTicket reads a ticket by ID.
func (s *Server) handleReadTicket(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input ReadTicketInput,
) (*mcp.CallToolResult, ReadTicketOutput, error) {
	if input.ID == "" {
		return nil, ReadTicketOutput{}, NewValidationError("id", "cannot be empty")
	}

	resp, err := s.sdkClient.GetTicketByID(input.ID)
	if err != nil {
		return nil, ReadTicketOutput{}, wrapSDKError(err)
	}

	out := ticketResponseToOutput(resp)

	if resp.HasConclusion {
		conclusion, err := s.sdkClient.GetConclusion(resp.ID)
		if err != nil {
			if s.config.Logger != nil {
				s.config.Logger.Warn("failed to fetch conclusion for ticket", "ticket_id", input.ID, "error", err)
			}
		} else {
			out.Conclusion = &ConclusionOutput{
				ID:          conclusion.ID,
				Body:        conclusion.Body,
				StartedAt:   conclusion.StartedAt.Format(time.RFC3339),
				ConcludedAt: conclusion.ConcludedAt.Format(time.RFC3339),
			}
		}
	}

	return nil, ReadTicketOutput{Ticket: out}, nil
}

// handleCreateWorkTicket creates a new work ticket via the daemon HTTP API.
func (s *Server) handleCreateWorkTicket(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input CreateWorkTicketInput,
) (*mcp.CallToolResult, CreateTicketOutput, error) {
	if input.Title == "" {
		return nil, CreateTicketOutput{}, NewValidationError("title", "is required")
	}
	if input.Repo == "" {
		return nil, CreateTicketOutput{}, NewValidationError("repo", "is required for work tickets")
	}

	// Parse dueDate if provided
	var dueDate *time.Time
	if input.DueDate != "" {
		parsed, err := time.Parse(time.RFC3339, input.DueDate)
		if err != nil {
			return nil, CreateTicketOutput{}, NewValidationError("due_date", "must be in RFC3339 format")
		}
		dueDate = &parsed
	}

	resp, err := s.sdkClient.CreateTicket(input.Title, input.Body, input.Repo, dueDate, input.References)
	if err != nil {
		return nil, CreateTicketOutput{}, wrapSDKError(err)
	}

	return nil, CreateTicketOutput{
		Ticket: ticketResponseToMetadataOutput(resp),
	}, nil
}

// handleUpdateTicket updates a ticket's mutable fields via the daemon HTTP API.
func (s *Server) handleUpdateTicket(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input UpdateTicketInput,
) (*mcp.CallToolResult, UpdateTicketOutput, error) {
	if input.ID == "" {
		return nil, UpdateTicketOutput{}, NewValidationError("id", "cannot be empty")
	}

	var (
		resp *sdk.TicketResponse
		err  error
	)

	if input.Title != nil || input.Body != nil || input.References != nil {
		resp, err = s.sdkClient.UpdateTicket(input.ID, input.Title, input.Body, input.References)
		if err != nil {
			return nil, UpdateTicketOutput{}, wrapSDKError(err)
		}
	}

	if input.DueDate != nil {
		if *input.DueDate == "" {
			resp, err = s.sdkClient.ClearDueDate(input.ID)
		} else {
			dueDate, parseErr := time.Parse(time.RFC3339, *input.DueDate)
			if parseErr != nil {
				return nil, UpdateTicketOutput{}, NewValidationError("dueDate", "must be empty or in RFC3339 format")
			}
			resp, err = s.sdkClient.SetDueDate(input.ID, dueDate)
		}
		if err != nil {
			return nil, UpdateTicketOutput{}, wrapSDKError(err)
		}
	}

	if resp == nil {
		resp, err = s.sdkClient.GetTicketByID(input.ID)
		if err != nil {
			return nil, UpdateTicketOutput{}, wrapSDKError(err)
		}
	}

	return nil, UpdateTicketOutput{
		Ticket: ticketResponseToOutput(resp),
	}, nil
}

func (s *Server) handleEditTicketBody(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input EditTicketBodyInput,
) (*mcp.CallToolResult, EditTicketBodyOutput, error) {
	if input.ID == "" {
		return nil, EditTicketBodyOutput{}, NewValidationError("id", "cannot be empty")
	}
	if input.OldString == "" {
		return nil, EditTicketBodyOutput{}, NewValidationError("oldString", "cannot be empty")
	}
	if input.OldString == input.NewString {
		return nil, EditTicketBodyOutput{}, NewValidationError("newString", "must differ from oldString")
	}

	current, err := s.sdkClient.GetTicketByID(input.ID)
	if err != nil {
		return nil, EditTicketBodyOutput{}, wrapSDKError(err)
	}

	replaced := 1
	if input.ReplaceAll {
		replaced = ticket.CountBodyEditMatches(current.Body, input.OldString)
	}

	resp, err := s.sdkClient.EditTicketBody(input.ID, input.OldString, input.NewString, input.ReplaceAll)
	if err != nil {
		return nil, EditTicketBodyOutput{}, wrapSDKError(err)
	}

	return nil, EditTicketBodyOutput{
		Success:  true,
		ID:       resp.ID,
		Replaced: replaced,
	}, nil
}

// handleDeleteTicket deletes a ticket via the daemon HTTP API.
func (s *Server) handleDeleteTicket(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input DeleteTicketInput,
) (*mcp.CallToolResult, DeleteTicketOutput, error) {
	if input.ID == "" {
		return nil, DeleteTicketOutput{}, NewValidationError("id", "cannot be empty")
	}

	err := s.sdkClient.DeleteTicket(input.ID)
	if err != nil {
		return nil, DeleteTicketOutput{}, wrapSDKError(err)
	}

	return nil, DeleteTicketOutput{
		Success: true,
		ID:      input.ID,
	}, nil
}

// handleMoveTicket moves a ticket to a different status via the daemon HTTP API.
func (s *Server) handleMoveTicket(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input MoveTicketInput,
) (*mcp.CallToolResult, MoveTicketOutput, error) {
	if input.ID == "" {
		return nil, MoveTicketOutput{}, NewValidationError("id", "cannot be empty")
	}
	if input.Status == "" {
		return nil, MoveTicketOutput{}, NewValidationError("status", "cannot be empty")
	}

	// Validate status
	if input.Status != "backlog" && input.Status != "progress" && input.Status != "done" {
		return nil, MoveTicketOutput{}, NewValidationError("status", "must be backlog, progress, or done")
	}

	_, err := s.sdkClient.MoveTicket(input.ID, input.Status)
	if err != nil {
		return nil, MoveTicketOutput{}, wrapSDKError(err)
	}

	return nil, MoveTicketOutput{
		Success: true,
		ID:      input.ID,
		Status:  input.Status,
	}, nil
}

// handleSpawnSession spawns a new agent session for a ticket.
// Delegates to the daemon HTTP API instead of calling spawn.Orchestrate() directly.
func (s *Server) handleSpawnSession(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input SpawnSessionInput,
) (*mcp.CallToolResult, SpawnSessionOutput, error) {
	if input.TicketID == "" {
		return nil, SpawnSessionOutput{}, NewValidationError("ticket_id", "cannot be empty")
	}
	if input.Variant == "" {
		return nil, SpawnSessionOutput{}, NewValidationError("variant", "is required — use listVariants to see available names")
	}

	// Validate mode if provided
	if input.Mode != "" && input.Mode != "normal" && input.Mode != "resume" && input.Mode != "fresh" {
		return nil, SpawnSessionOutput{}, NewValidationError("mode", "must be 'normal', 'resume', or 'fresh'")
	}

	projectPath := s.config.ArchitectPath

	// Look up ticket status via daemon API (needed to build the spawn URL)
	ticketResp, err := s.sdkClient.GetTicketByID(input.TicketID)
	if err != nil {
		return nil, SpawnSessionOutput{}, wrapSDKError(err)
	}

	// Build HTTP request to daemon
	url := fmt.Sprintf("%s/tickets/%s/%s/spawn?variant=%s", s.config.DaemonURL, ticketResp.Status, input.TicketID, input.Variant)
	if input.Mode != "" {
		url += "&mode=" + input.Mode
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return nil, SpawnSessionOutput{}, NewInternalError("failed to create request: " + err.Error())
	}
	httpReq.Header.Set("X-Cortex-Architect", projectPath)

	// Execute request
	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, SpawnSessionOutput{}, NewInternalError("failed to contact daemon: " + err.Error())
	}
	defer func() { _ = resp.Body.Close() }()

	// Map response
	switch resp.StatusCode {
	case http.StatusCreated: // 201 - spawned/resumed
		var spawnResp api.SpawnResponse
		if err := json.NewDecoder(resp.Body).Decode(&spawnResp); err != nil {
			return nil, SpawnSessionOutput{}, NewInternalError("failed to decode response: " + err.Error())
		}
		return nil, SpawnSessionOutput{
			Success:    true,
			TicketID:   input.TicketID,
			TmuxWindow: spawnResp.Session.TmuxWindow,
		}, nil

	case http.StatusOK: // 200 - already active
		return nil, SpawnSessionOutput{State: "active"}, NewStateConflictError("active", input.Mode, "session is currently active - wait for it to finish or close the tmux window")

	case http.StatusConflict: // 409 - state/orphaned error
		var errResp types.ErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
			return nil, SpawnSessionOutput{}, NewInternalError("failed to decode error response: " + err.Error())
		}
		state := parseStateFromError(errResp.Code, errResp.Error)
		return nil, SpawnSessionOutput{State: state}, NewStateConflictError(state, input.Mode, errResp.Error)

	case http.StatusNotFound: // 404 - ticket not found
		return nil, SpawnSessionOutput{}, NewNotFoundError("ticket", input.TicketID)

	case http.StatusBadRequest: // 400 - config error
		var errResp types.ErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
			return nil, SpawnSessionOutput{}, NewInternalError("failed to decode error response: " + err.Error())
		}
		return nil, SpawnSessionOutput{
			Success: false,
			Message: errResp.Error,
		}, nil

	case http.StatusServiceUnavailable: // 503 - tmux unavailable
		var errResp types.ErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
			return nil, SpawnSessionOutput{}, NewInternalError("failed to decode error response: " + err.Error())
		}
		return nil, SpawnSessionOutput{
			Success: false,
			Message: errResp.Error,
		}, nil

	default: // 500, etc.
		var errResp types.ErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
			return nil, SpawnSessionOutput{
				Success: false,
				Message: fmt.Sprintf("daemon returned status %d", resp.StatusCode),
			}, nil
		}
		return nil, SpawnSessionOutput{
			Success: false,
			Message: errResp.Error,
		}, nil
	}
}

// parseStateFromError extracts the session state from an HTTP error response.
func parseStateFromError(code, message string) string {
	if code == "session_orphaned" {
		return "orphaned"
	}
	// For state_conflict, extract state from error message.
	// Format: "spawn: ticket <id> in state <state>: <message>"
	if idx := strings.Index(message, "in state "); idx != -1 {
		rest := message[idx+len("in state "):]
		if end := strings.Index(rest, ":"); end != -1 {
			return rest[:end]
		}
	}
	return "unknown"
}

// handleListConclusions lists persistent conclusion records (metadata only, no body).
func (s *Server) handleListConclusions(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input ListConclusionsInput,
) (*mcp.CallToolResult, ListConclusionsOutput, error) {
	limit := input.Limit
	if limit == 0 {
		limit = 10
	}

	resp, err := s.sdkClient.ListConclusions(sdk.ListConclusionsParams{
		Type:   input.Type,
		Limit:  limit,
		Offset: input.Offset,
	})
	if err != nil {
		return nil, ListConclusionsOutput{}, wrapSDKError(err)
	}

	items := make([]ConclusionListItem, len(resp.Conclusions))
	for i, c := range resp.Conclusions {
		items[i] = ConclusionListItem{
			ID:          c.ID,
			TicketID:    c.TicketID,
			CollabID:    c.CollabID,
			Agent:       c.Agent,
			Profile:     c.Profile,
			StartedAt:   c.StartedAt.Format(time.RFC3339),
			ConcludedAt: c.ConcludedAt.Format(time.RFC3339),
			Rejected:    c.Rejected,
		}
	}

	return nil, ListConclusionsOutput{
		Conclusions: items,
		Total:       resp.Total,
	}, nil
}

// handleReadConclusion reads a conclusion record by ID, including full body.
func (s *Server) handleReadConclusion(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input ReadConclusionInput,
) (*mcp.CallToolResult, ReadConclusionOutput, error) {
	if input.ID == "" {
		return nil, ReadConclusionOutput{}, NewValidationError("id", "cannot be empty")
	}

	resp, err := s.sdkClient.GetConclusion(input.ID)
	if err != nil {
		return nil, ReadConclusionOutput{}, wrapSDKError(err)
	}

	return nil, ReadConclusionOutput{
		Conclusion: ConclusionOutput{
			ID:          resp.ID,
			Body:        resp.Body,
			StartedAt:   resp.StartedAt.Format(time.RFC3339),
			ConcludedAt: resp.ConcludedAt.Format(time.RFC3339),
		},
	}, nil
}

// handleSpawnCollabSession spawns a collab session via the daemon HTTP API.
func (s *Server) handleSpawnCollabSession(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input SpawnCollabSessionInput,
) (*mcp.CallToolResult, SpawnCollabSessionOutput, error) {
	if input.Path == "" {
		return nil, SpawnCollabSessionOutput{}, NewValidationError("path", "cannot be empty")
	}
	if input.Prompt == "" {
		return nil, SpawnCollabSessionOutput{}, NewValidationError("prompt", "cannot be empty")
	}
	if input.Slug == "" {
		return nil, SpawnCollabSessionOutput{}, NewValidationError("slug", "is required")
	}
	if input.Variant == "" {
		return nil, SpawnCollabSessionOutput{}, NewValidationError("variant", "is required — use listVariants to see available names")
	}

	resp, err := s.sdkClient.SpawnCollabSession(input.Path, input.Prompt, input.Slug, "normal", input.Variant)
	if err != nil {
		return nil, SpawnCollabSessionOutput{}, wrapSDKError(err)
	}

	return nil, SpawnCollabSessionOutput{
		Success:    true,
		CollabID:   resp.CollabID,
		TmuxWindow: resp.TmuxWindow,
		State:      resp.State,
	}, nil
}

// handleArchitectConcludeSession concludes the architect session.
func (s *Server) handleArchitectConcludeSession(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input ConcludeSessionInput,
) (*mcp.CallToolResult, ArchitectConcludeOutput, error) {
	if input.Body == "" {
		return nil, ArchitectConcludeOutput{}, NewValidationError("body", "cannot be empty")
	}

	startedAt := os.Getenv("CORTEX_STARTED_AT")
	resp, err := s.sdkClient.ConcludeArchitectSession(input.Body, startedAt)
	if err != nil {
		return nil, ArchitectConcludeOutput{}, wrapSDKError(err)
	}

	return nil, ArchitectConcludeOutput{
		Success: resp.Success,
		Message: resp.Message,
	}, nil
}

// handleListVariants returns the available agent variant names via the daemon HTTP API.
func (s *Server) handleListVariants(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input ListVariantsInput,
) (*mcp.CallToolResult, ListVariantsOutput, error) {
	variants, err := s.sdkClient.GetVariants()
	if err != nil {
		return nil, ListVariantsOutput{}, wrapSDKError(err)
	}
	return nil, ListVariantsOutput{Variants: variants}, nil
}

// handleCreateFollowUpTicket creates a follow-up work ticket.
// For ticket sessions (s.session.TicketID != ""), it auto-links bidirectionally
// with the originating ticket. For collab sessions, it creates the ticket with no references.
func (s *Server) handleCreateFollowUpTicket(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input CreateFollowUpTicketInput,
) (*mcp.CallToolResult, CreateFollowUpTicketOutput, error) {
	if input.Title == "" {
		return nil, CreateFollowUpTicketOutput{}, NewValidationError("title", "is required")
	}
	if input.Repo == "" {
		return nil, CreateFollowUpTicketOutput{}, NewValidationError("repo", "is required for work tickets")
	}

	var dueDate *time.Time
	if input.DueDate != "" {
		parsed, err := time.Parse(time.RFC3339, input.DueDate)
		if err != nil {
			return nil, CreateFollowUpTicketOutput{}, NewValidationError("due_date", "must be in RFC3339 format")
		}
		dueDate = &parsed
	}

	originID := s.session.TicketID
	var references []string
	if originID != "" {
		references = []string{originID}
	}

	resp, err := s.sdkClient.CreateTicket(input.Title, input.Body, input.Repo, dueDate, references)
	if err != nil {
		return nil, CreateFollowUpTicketOutput{}, wrapSDKError(err)
	}

	newID := resp.ID

	if originID != "" {
		origin, err := s.sdkClient.GetTicketByID(originID)
		if err != nil {
			return nil, CreateFollowUpTicketOutput{}, NewInternalError(fmt.Sprintf(
				"follow-up ticket %s was created but failed to fetch origin ticket %s: %s",
				newID, originID, err,
			))
		}

		updatedRefs := append(origin.References, newID)
		if _, err := s.sdkClient.UpdateTicket(originID, nil, nil, &updatedRefs); err != nil {
			return nil, CreateFollowUpTicketOutput{}, NewInternalError(fmt.Sprintf(
				"follow-up ticket %s was created but failed to update origin ticket %s references: %s",
				newID, originID, err,
			))
		}
	}

	return nil, CreateFollowUpTicketOutput{
		Ticket: ticketResponseToOutput(resp),
	}, nil
}

// handleSearch searches across all tickets (all statuses) and conclusions.
func (s *Server) handleSearch(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input SearchInput,
) (*mcp.CallToolResult, SearchOutput, error) {
	if input.Query == "" {
		return nil, SearchOutput{}, NewValidationError("query", "cannot be empty")
	}

	limit := 25
	if input.Limit > 0 {
		limit = input.Limit
	}

	// 1. All tickets matching query (title+body, server-side filtered).
	allTickets, err := s.sdkClient.ListAllTickets(input.Query, nil)
	if err != nil {
		return nil, SearchOutput{}, wrapSDKError(err)
	}

	// 2. All conclusions whose body matches the query (server-side filtered).
	allConclusions, err := s.sdkClient.ListConclusions(sdk.ListConclusionsParams{Query: input.Query, Limit: 0})
	if err != nil {
		return nil, SearchOutput{}, wrapSDKError(err)
	}

	// 3. Build a unique set of ticket IDs to fetch — start from direct ticket matches.
	ticketIDSet := make(map[string]bool)
	for _, t := range allTickets.Backlog {
		ticketIDSet[t.ID] = true
	}
	for _, t := range allTickets.Progress {
		ticketIDSet[t.ID] = true
	}
	for _, t := range allTickets.Done {
		ticketIDSet[t.ID] = true
	}

	// 4. Process conclusion matches: ticketed ones add parent; ticketless are bare conclusions.
	var bareConclusions []types.ConclusionSummary
	for _, c := range allConclusions.Conclusions {
		if c.TicketID != "" {
			ticketIDSet[c.TicketID] = true
		} else {
			bareConclusions = append(bareConclusions, c)
		}
	}

	// resultWithTime pairs a SearchResultItem with a timestamp for sorting.
	type resultWithTime struct {
		item SearchResultItem
		t    time.Time
	}
	var combined []resultWithTime

	// 5. Fetch full ticket details and nest conclusion when available.
	for ticketID := range ticketIDSet {
		resp, err := s.sdkClient.GetTicketByID(ticketID)
		if err != nil {
			// Ticket may have been deleted between listing and fetching — skip.
			continue
		}
		out := ticketResponseToOutput(resp)
		if resp.HasConclusion {
			conclusion, err := s.sdkClient.GetConclusion(resp.ID)
			if err == nil {
				out.Conclusion = &ConclusionOutput{
					ID:          conclusion.ID,
					Body:        conclusion.Body,
					StartedAt:   conclusion.StartedAt.Format(time.RFC3339),
					ConcludedAt: conclusion.ConcludedAt.Format(time.RFC3339),
				}
			}
		}
		combined = append(combined, resultWithTime{
			item: SearchResultItem{Ticket: &out},
			t:    resp.Updated,
		})
	}

	// 6. Fetch full bodies for bare ticketless conclusions.
	for _, bc := range bareConclusions {
		resp, err := s.sdkClient.GetConclusion(bc.ID)
		if err != nil {
			continue
		}
		cOut := ConclusionOutput{
			ID:          resp.ID,
			Body:        resp.Body,
			StartedAt:   resp.StartedAt.Format(time.RFC3339),
			ConcludedAt: resp.ConcludedAt.Format(time.RFC3339),
		}
		combined = append(combined, resultWithTime{
			item: SearchResultItem{Conclusion: &cOut},
			t:    resp.ConcludedAt,
		})
	}

	// 7. Sort newest-updated first.
	sort.Slice(combined, func(i, j int) bool {
		return combined[i].t.After(combined[j].t)
	})

	// 8. Apply limit.
	total := len(combined)
	if len(combined) > limit {
		combined = combined[:limit]
	}

	results := make([]SearchResultItem, len(combined))
	for i, r := range combined {
		results[i] = r.item
	}

	return nil, SearchOutput{Results: results, Total: total}, nil
}
