package mcp

import (
	"context"
	"log"

	"github.com/kareemaly/cortex1/internal/lifecycle"
	"github.com/kareemaly/cortex1/internal/ticket"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// registerTicketTools registers all tools available to ticket sessions.
func (s *Server) registerTicketTools() {
	// Read own ticket (no input needed, uses session ticket ID)
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "readTicket",
		Description: "Read your assigned ticket details",
	}, s.handleReadOwnTicket)

	// Pickup ticket (move to in_progress)
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "pickupTicket",
		Description: "Start working on your assigned ticket (moves to in_progress)",
	}, s.handlePickupTicket)

	// Submit report
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "submitReport",
		Description: "Update your session report with files, decisions, and summary",
	}, s.handleSubmitReport)

	// Approve (end session and move to done)
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "approve",
		Description: "Mark your work as complete and move ticket to done",
	}, s.handleApprove)
}

// EmptyInput is used for tools that don't require input.
type EmptyInput struct{}

// handleReadOwnTicket reads the ticket assigned to this session.
func (s *Server) handleReadOwnTicket(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input EmptyInput,
) (*mcp.CallToolResult, ReadTicketOutput, error) {
	t, status, err := s.store.Get(s.session.TicketID)
	if err != nil {
		return nil, ReadTicketOutput{}, WrapTicketError(err)
	}

	return nil, ReadTicketOutput{
		Ticket: ToTicketOutput(t, status),
	}, nil
}

// handlePickupTicket moves the assigned ticket to in_progress status.
func (s *Server) handlePickupTicket(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input EmptyInput,
) (*mcp.CallToolResult, PickupTicketOutput, error) {
	// Check if ticket exists
	t, currentStatus, err := s.store.Get(s.session.TicketID)
	if err != nil {
		return nil, PickupTicketOutput{}, WrapTicketError(err)
	}

	// Check if already in progress
	if currentStatus == ticket.StatusProgress {
		return nil, PickupTicketOutput{
			Success: true,
			Message: "Ticket is already in progress",
		}, nil
	}

	// Check if done
	if currentStatus == ticket.StatusDone {
		return nil, PickupTicketOutput{}, NewValidationError("status", "cannot pickup a completed ticket")
	}

	// Move to in_progress
	err = s.store.Move(s.session.TicketID, ticket.StatusProgress)
	if err != nil {
		return nil, PickupTicketOutput{}, WrapTicketError(err)
	}

	// Execute on_pickup hooks (failures are logged but don't fail the operation)
	var hooksOutput *HooksExecutionOutput
	hooks := s.getHooksForType(lifecycle.HookOnPickup)
	if len(hooks) > 0 && s.config.ProjectPath != "" {
		vars := buildTemplateVars(t)
		result, err := s.lifecycle.Execute(ctx, s.config.ProjectPath, lifecycle.HookOnPickup, hooks, vars)
		if err != nil {
			log.Printf("on_pickup hooks execution error: %v", err)
		} else {
			hooksOutput = convertExecutionResult(result)
			if !result.Success {
				log.Printf("on_pickup hooks failed (exit code non-zero)")
			}
		}
	}

	return nil, PickupTicketOutput{
		Success: true,
		Message: "Ticket moved to in_progress",
		Hooks:   hooksOutput,
	}, nil
}

// handleSubmitReport updates the session report for the assigned ticket.
func (s *Server) handleSubmitReport(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input SubmitReportInput,
) (*mcp.CallToolResult, SubmitReportOutput, error) {
	// Get the ticket to find the active session
	t, _, err := s.store.Get(s.session.TicketID)
	if err != nil {
		return nil, SubmitReportOutput{}, WrapTicketError(err)
	}

	// Find active session
	var activeSessionID string
	for _, sess := range t.Sessions {
		if sess.IsActive() {
			activeSessionID = sess.ID
			break
		}
	}

	if activeSessionID == "" {
		return nil, SubmitReportOutput{}, NewValidationError("session", "no active session found")
	}

	// Build report
	report := ticket.Report{
		Files:        input.Files,
		ScopeChanges: input.ScopeChanges,
		Decisions:    input.Decisions,
		Summary:      input.Summary,
	}

	// Handle nil slices
	if report.Files == nil {
		report.Files = []string{}
	}
	if report.Decisions == nil {
		report.Decisions = []string{}
	}

	// Update the report
	err = s.store.UpdateSessionReport(s.session.TicketID, activeSessionID, report)
	if err != nil {
		return nil, SubmitReportOutput{}, WrapTicketError(err)
	}

	// Execute on_submit hooks (failures are logged but don't fail the operation)
	var hooksOutput *HooksExecutionOutput
	hooks := s.getHooksForType(lifecycle.HookOnSubmit)
	if len(hooks) > 0 && s.config.ProjectPath != "" {
		vars := buildTemplateVars(t)
		result, err := s.lifecycle.Execute(ctx, s.config.ProjectPath, lifecycle.HookOnSubmit, hooks, vars)
		if err != nil {
			log.Printf("on_submit hooks execution error: %v", err)
		} else {
			hooksOutput = convertExecutionResult(result)
			if !result.Success {
				log.Printf("on_submit hooks failed (exit code non-zero)")
			}
		}
	}

	return nil, SubmitReportOutput{
		Success: true,
		Report: ReportOutput{
			Files:        report.Files,
			ScopeChanges: report.ScopeChanges,
			Decisions:    report.Decisions,
			Summary:      report.Summary,
		},
		Hooks: hooksOutput,
	}, nil
}

// handleApprove ends the active session and moves the ticket to done.
func (s *Server) handleApprove(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input ApproveInput,
) (*mcp.CallToolResult, ApproveOutput, error) {
	// Get the ticket to find the active session
	t, currentStatus, err := s.store.Get(s.session.TicketID)
	if err != nil {
		return nil, ApproveOutput{}, WrapTicketError(err)
	}

	// Find active session
	var activeSessionID string
	for _, sess := range t.Sessions {
		if sess.IsActive() {
			activeSessionID = sess.ID
			break
		}
	}

	if activeSessionID == "" {
		return nil, ApproveOutput{}, NewValidationError("session", "no active session found")
	}

	// Update summary if provided
	if input.Summary != "" {
		// Get current report
		var currentReport ticket.Report
		for _, sess := range t.Sessions {
			if sess.ID == activeSessionID {
				currentReport = sess.Report
				break
			}
		}
		currentReport.Summary = input.Summary
		err = s.store.UpdateSessionReport(s.session.TicketID, activeSessionID, currentReport)
		if err != nil {
			return nil, ApproveOutput{}, WrapTicketError(err)
		}
	}

	// Execute on_approve hooks - MUST succeed before moving to done
	var hooksOutput *HooksExecutionOutput
	hooks := s.getHooksForType(lifecycle.HookOnApprove)
	if len(hooks) > 0 && s.config.ProjectPath != "" {
		vars := buildTemplateVars(t).WithCommitMessage(input.CommitMessage)
		result, err := s.lifecycle.Execute(ctx, s.config.ProjectPath, lifecycle.HookOnApprove, hooks, vars)
		if err != nil {
			log.Printf("on_approve hooks execution error: %v", err)
			return nil, ApproveOutput{
				Success:  false,
				TicketID: s.session.TicketID,
				Status:   string(currentStatus),
				Message:  "on_approve hooks failed to execute: " + err.Error(),
			}, nil
		}
		hooksOutput = convertExecutionResult(result)
		if !result.Success {
			// Hooks failed - keep ticket in progress
			return nil, ApproveOutput{
				Success:  false,
				TicketID: s.session.TicketID,
				Status:   string(currentStatus),
				Message:  "on_approve hooks failed (non-zero exit code)",
				Hooks:    hooksOutput,
			}, nil
		}
	}

	// End the session
	err = s.store.EndSession(s.session.TicketID, activeSessionID)
	if err != nil {
		return nil, ApproveOutput{}, WrapTicketError(err)
	}

	// Move ticket to done
	err = s.store.Move(s.session.TicketID, ticket.StatusDone)
	if err != nil {
		return nil, ApproveOutput{}, WrapTicketError(err)
	}

	return nil, ApproveOutput{
		Success:  true,
		TicketID: s.session.TicketID,
		Status:   string(ticket.StatusDone),
		Hooks:    hooksOutput,
	}, nil
}
