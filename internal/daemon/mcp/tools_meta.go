package mcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// registerMetaTools registers all tools available to meta sessions.
func (s *Server) registerMetaTools() {
	// Project management
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "listProjects",
		Description: "List all registered projects with their paths, titles, and ticket counts.",
	}, s.handleListProjects)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "registerProject",
		Description: "Register a project directory with the Cortex daemon. The directory must contain a .cortex/ folder (run 'cortex init' first).",
	}, s.handleRegisterProject)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "unregisterProject",
		Description: "Remove a project from the Cortex registry. Does not delete any files.",
	}, s.handleUnregisterProject)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "spawnArchitect",
		Description: "Spawn an architect session for a registered project.",
	}, s.handleMetaSpawnArchitect)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "listSessions",
		Description: "List all active agent sessions for a project.",
	}, s.handleMetaListSessions)

	// Configuration
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "readProjectConfig",
		Description: "Read a project's cortex.yaml configuration file.",
	}, s.handleReadProjectConfig)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "updateProjectConfig",
		Description: "Update a project's cortex.yaml configuration. Provide the full YAML content.",
	}, s.handleUpdateProjectConfig)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "readGlobalConfig",
		Description: "Read the global Cortex configuration (~/.cortex/settings.yaml).",
	}, s.handleReadGlobalConfig)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "updateGlobalConfig",
		Description: "Update the global Cortex configuration. Provide the full YAML content.",
	}, s.handleUpdateGlobalConfig)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "readPrompt",
		Description: "Read a prompt template for a given role and stage. Returns the resolved content with source path.",
	}, s.handleReadPrompt)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "updatePrompt",
		Description: "Update a prompt template. This auto-ejects the prompt to the project's .cortex/prompts/ directory.",
	}, s.handleUpdatePrompt)

	// Debugging
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "readDaemonLogs",
		Description: "Read recent daemon log output. Optionally filter by log level.",
	}, s.handleReadDaemonLogs)

	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "daemonStatus",
		Description: "Get daemon status including uptime, version, and project count.",
	}, s.handleDaemonStatus)

	// Session management
	mcp.AddTool(s.mcpServer, &mcp.Tool{
		Name:        "concludeSession",
		Description: "Conclude the meta session and clean up.",
	}, s.handleMetaConcludeSession)
}

// handleRegisterProject registers a project with the daemon.
func (s *Server) handleRegisterProject(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input RegisterProjectInput,
) (*mcp.CallToolResult, map[string]any, error) {
	if input.Path == "" {
		return nil, nil, NewValidationError("path", "cannot be empty")
	}

	// Call POST /projects on the daemon
	body := map[string]string{"path": input.Path}
	if input.Title != "" {
		body["title"] = input.Title
	}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, nil, NewInternalError("failed to encode request: " + err.Error())
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, s.config.DaemonURL+"/projects", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, nil, NewInternalError("failed to create request: " + err.Error())
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, nil, NewInternalError("failed to contact daemon: " + err.Error())
	}
	defer func() { _ = resp.Body.Close() }()

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, nil, NewInternalError("failed to decode response: " + err.Error())
	}

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		msg := "failed to register project"
		if m, ok := result["error"].(string); ok {
			msg = m
		}
		return nil, nil, NewInternalError(msg)
	}

	return nil, result, nil
}

// handleUnregisterProject removes a project from the registry.
func (s *Server) handleUnregisterProject(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input UnregisterProjectInput,
) (*mcp.CallToolResult, map[string]any, error) {
	if input.Path == "" {
		return nil, nil, NewValidationError("path", "cannot be empty")
	}

	err := s.sdkClient.UnlinkProject(input.Path)
	if err != nil {
		return nil, nil, wrapSDKError(err)
	}

	return nil, map[string]any{
		"success": true,
		"path":    input.Path,
	}, nil
}

// handleMetaSpawnArchitect spawns an architect for a specific project.
func (s *Server) handleMetaSpawnArchitect(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input SpawnArchitectInput,
) (*mcp.CallToolResult, map[string]any, error) {
	if input.ProjectPath == "" {
		return nil, nil, NewValidationError("project_path", "cannot be empty")
	}

	// Validate project is registered
	if err := s.validateProjectPath(input.ProjectPath); err != nil {
		return nil, nil, err
	}

	// Validate mode
	if input.Mode != "" && input.Mode != "normal" && input.Mode != "resume" && input.Mode != "fresh" {
		return nil, nil, NewValidationError("mode", "must be 'normal', 'resume', or 'fresh'")
	}

	// Call the architect spawn endpoint
	url := s.config.DaemonURL + "/architect/spawn"
	if input.Mode != "" {
		url += "?mode=" + input.Mode
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return nil, nil, NewInternalError("failed to create request: " + err.Error())
	}
	httpReq.Header.Set("X-Cortex-Project", input.ProjectPath)

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, nil, NewInternalError("failed to contact daemon: " + err.Error())
	}
	defer func() { _ = resp.Body.Close() }()

	var result map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, nil, NewInternalError("failed to decode response: " + err.Error())
	}

	if resp.StatusCode == http.StatusConflict {
		msg := "architect session conflict"
		if m, ok := result["error"].(string); ok {
			msg = m
		}
		code, _ := result["code"].(string)
		state := parseStateFromError(code, msg)
		return nil, nil, NewStateConflictError(state, input.Mode, msg)
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
		msg := fmt.Sprintf("daemon returned status %d", resp.StatusCode)
		if m, ok := result["error"].(string); ok {
			msg = m
		}
		return nil, nil, NewInternalError(msg)
	}

	return nil, result, nil
}

// handleMetaListSessions lists sessions for a project from the meta context.
func (s *Server) handleMetaListSessions(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input ListSessionsInput,
) (*mcp.CallToolResult, ListSessionsOutput, error) {
	if input.ProjectPath == "" {
		return nil, ListSessionsOutput{}, NewValidationError("project_path", "cannot be empty")
	}

	if err := s.validateProjectPath(input.ProjectPath); err != nil {
		return nil, ListSessionsOutput{}, err
	}

	client := s.getClientForProject(input.ProjectPath)

	resp, err := client.ListSessions()
	if err != nil {
		return nil, ListSessionsOutput{}, wrapSDKError(err)
	}

	items := make([]SessionListItem, len(resp.Sessions))
	for i, sess := range resp.Sessions {
		items[i] = SessionListItem{
			SessionID:   sess.SessionID,
			SessionType: sess.SessionType,
			TicketID:    sess.TicketID,
			TicketTitle: sess.TicketTitle,
			Agent:       sess.Agent,
			TmuxWindow:  sess.TmuxWindow,
			StartedAt:   sess.StartedAt,
			Status:      sess.Status,
			Tool:        sess.Tool,
		}
	}

	return nil, ListSessionsOutput{
		Sessions: items,
		Total:    len(items),
	}, nil
}

// handleReadProjectConfig reads a project's cortex.yaml.
func (s *Server) handleReadProjectConfig(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input ReadProjectConfigInput,
) (*mcp.CallToolResult, ConfigOutput, error) {
	if input.ProjectPath == "" {
		return nil, ConfigOutput{}, NewValidationError("project_path", "cannot be empty")
	}

	// Call the daemon endpoint
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, s.config.DaemonURL+"/config/project", nil)
	if err != nil {
		return nil, ConfigOutput{}, NewInternalError("failed to create request: " + err.Error())
	}
	httpReq.Header.Set("X-Cortex-Project", input.ProjectPath)

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, ConfigOutput{}, NewInternalError("failed to contact daemon: " + err.Error())
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, ConfigOutput{}, wrapHTTPError(resp)
	}

	var result struct {
		Content string `json:"content"`
		Path    string `json:"path"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, ConfigOutput{}, NewInternalError("failed to decode response: " + err.Error())
	}

	return nil, ConfigOutput{
		Content: result.Content,
		Path:    result.Path,
	}, nil
}

// handleUpdateProjectConfig updates a project's cortex.yaml.
func (s *Server) handleUpdateProjectConfig(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input UpdateProjectConfigInput,
) (*mcp.CallToolResult, ConfigOutput, error) {
	if input.ProjectPath == "" {
		return nil, ConfigOutput{}, NewValidationError("project_path", "cannot be empty")
	}
	if input.Content == "" {
		return nil, ConfigOutput{}, NewValidationError("content", "cannot be empty")
	}

	body := map[string]string{"content": input.Content}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, ConfigOutput{}, NewInternalError("failed to encode request: " + err.Error())
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPut, s.config.DaemonURL+"/config/project", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, ConfigOutput{}, NewInternalError("failed to create request: " + err.Error())
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Cortex-Project", input.ProjectPath)

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, ConfigOutput{}, NewInternalError("failed to contact daemon: " + err.Error())
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, ConfigOutput{}, wrapHTTPError(resp)
	}

	var result struct {
		Content string `json:"content"`
		Path    string `json:"path"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, ConfigOutput{}, NewInternalError("failed to decode response: " + err.Error())
	}

	return nil, ConfigOutput{
		Content: result.Content,
		Path:    result.Path,
	}, nil
}

// handleReadGlobalConfig reads the global settings.yaml.
func (s *Server) handleReadGlobalConfig(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input ReadGlobalConfigInput,
) (*mcp.CallToolResult, ConfigOutput, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, s.config.DaemonURL+"/config/global", nil)
	if err != nil {
		return nil, ConfigOutput{}, NewInternalError("failed to create request: " + err.Error())
	}

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, ConfigOutput{}, NewInternalError("failed to contact daemon: " + err.Error())
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, ConfigOutput{}, wrapHTTPError(resp)
	}

	var result struct {
		Content string `json:"content"`
		Path    string `json:"path"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, ConfigOutput{}, NewInternalError("failed to decode response: " + err.Error())
	}

	return nil, ConfigOutput{
		Content: result.Content,
		Path:    result.Path,
	}, nil
}

// handleUpdateGlobalConfig updates the global settings.yaml.
func (s *Server) handleUpdateGlobalConfig(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input UpdateGlobalConfigInput,
) (*mcp.CallToolResult, ConfigOutput, error) {
	if input.Content == "" {
		return nil, ConfigOutput{}, NewValidationError("content", "cannot be empty")
	}

	body := map[string]string{"content": input.Content}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, ConfigOutput{}, NewInternalError("failed to encode request: " + err.Error())
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPut, s.config.DaemonURL+"/config/global", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, ConfigOutput{}, NewInternalError("failed to create request: " + err.Error())
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, ConfigOutput{}, NewInternalError("failed to contact daemon: " + err.Error())
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, ConfigOutput{}, wrapHTTPError(resp)
	}

	var result struct {
		Content string `json:"content"`
		Path    string `json:"path"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, ConfigOutput{}, NewInternalError("failed to decode response: " + err.Error())
	}

	return nil, ConfigOutput{
		Content: result.Content,
		Path:    result.Path,
	}, nil
}

// handleReadPrompt reads a prompt template via the daemon API.
func (s *Server) handleReadPrompt(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input ReadPromptInput,
) (*mcp.CallToolResult, PromptOutput, error) {
	if input.ProjectPath == "" {
		return nil, PromptOutput{}, NewValidationError("project_path", "cannot be empty")
	}
	if input.Role == "" {
		return nil, PromptOutput{}, NewValidationError("role", "cannot be empty")
	}
	if input.Stage == "" {
		return nil, PromptOutput{}, NewValidationError("stage", "cannot be empty")
	}

	url := fmt.Sprintf("%s/prompts/resolve?role=%s&stage=%s", s.config.DaemonURL, input.Role, input.Stage)
	if input.TicketType != "" {
		url += "&type=" + input.TicketType
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, PromptOutput{}, NewInternalError("failed to create request: " + err.Error())
	}
	httpReq.Header.Set("X-Cortex-Project", input.ProjectPath)

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, PromptOutput{}, NewInternalError("failed to contact daemon: " + err.Error())
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, PromptOutput{}, wrapHTTPError(resp)
	}

	var result struct {
		Content    string `json:"content"`
		SourcePath string `json:"source_path"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, PromptOutput{}, NewInternalError("failed to decode response: " + err.Error())
	}

	return nil, PromptOutput{
		Content:    result.Content,
		SourcePath: result.SourcePath,
	}, nil
}

// handleUpdatePrompt updates a prompt by writing to the project's ejected prompt path.
func (s *Server) handleUpdatePrompt(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input UpdatePromptInput,
) (*mcp.CallToolResult, PromptOutput, error) {
	if input.ProjectPath == "" {
		return nil, PromptOutput{}, NewValidationError("project_path", "cannot be empty")
	}
	if input.Role == "" {
		return nil, PromptOutput{}, NewValidationError("role", "cannot be empty")
	}
	if input.Stage == "" {
		return nil, PromptOutput{}, NewValidationError("stage", "cannot be empty")
	}
	if input.Content == "" {
		return nil, PromptOutput{}, NewValidationError("content", "cannot be empty")
	}

	// Build the ejected prompt path
	// Pattern: {project}/.cortex/prompts/{role}/{STAGE}.md
	// For ticket types: {project}/.cortex/prompts/ticket/{type}/{STAGE}.md
	promptDir := input.ProjectPath + "/.cortex/prompts/" + input.Role
	if input.TicketType != "" {
		promptDir += "/" + input.TicketType
	}
	promptPath := promptDir + "/" + input.Stage + ".md"

	// Create directory and write the file
	if err := os.MkdirAll(promptDir, 0755); err != nil {
		return nil, PromptOutput{}, NewInternalError("failed to create prompt directory: " + err.Error())
	}

	if err := os.WriteFile(promptPath, []byte(input.Content), 0644); err != nil {
		return nil, PromptOutput{}, NewInternalError("failed to write prompt file: " + err.Error())
	}

	return nil, PromptOutput{
		Content:    input.Content,
		SourcePath: promptPath,
	}, nil
}

// handleReadDaemonLogs reads recent daemon logs.
func (s *Server) handleReadDaemonLogs(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input ReadDaemonLogsInput,
) (*mcp.CallToolResult, DaemonLogsOutput, error) {
	url := s.config.DaemonURL + "/daemon/logs"
	params := []string{}
	if input.Lines > 0 {
		params = append(params, fmt.Sprintf("lines=%d", input.Lines))
	}
	if input.Level != "" {
		params = append(params, "level="+input.Level)
	}
	if len(params) > 0 {
		url += "?" + joinParams(params)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, DaemonLogsOutput{}, NewInternalError("failed to create request: " + err.Error())
	}

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, DaemonLogsOutput{}, NewInternalError("failed to contact daemon: " + err.Error())
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, DaemonLogsOutput{}, wrapHTTPError(resp)
	}

	var result struct {
		Content string `json:"content"`
		Path    string `json:"path"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, DaemonLogsOutput{}, NewInternalError("failed to decode response: " + err.Error())
	}

	return nil, DaemonLogsOutput{
		Content: result.Content,
		Path:    result.Path,
	}, nil
}

// handleDaemonStatus returns daemon status.
func (s *Server) handleDaemonStatus(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input DaemonStatusInput,
) (*mcp.CallToolResult, DaemonStatusOutput, error) {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, s.config.DaemonURL+"/daemon/status", nil)
	if err != nil {
		return nil, DaemonStatusOutput{}, NewInternalError("failed to create request: " + err.Error())
	}

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, DaemonStatusOutput{}, NewInternalError("failed to contact daemon: " + err.Error())
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, DaemonStatusOutput{}, wrapHTTPError(resp)
	}

	var result DaemonStatusOutput
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, DaemonStatusOutput{}, NewInternalError("failed to decode response: " + err.Error())
	}

	return nil, result, nil
}

// handleMetaConcludeSession concludes the meta session.
func (s *Server) handleMetaConcludeSession(
	ctx context.Context,
	req *mcp.CallToolRequest,
	input MetaConcludeSessionInput,
) (*mcp.CallToolResult, ArchitectConcludeOutput, error) {
	if input.Content == "" {
		return nil, ArchitectConcludeOutput{}, NewValidationError("content", "cannot be empty")
	}

	// Call POST /meta/conclude
	body := map[string]string{"content": input.Content}
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, ArchitectConcludeOutput{}, NewInternalError("failed to encode request: " + err.Error())
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, s.config.DaemonURL+"/meta/conclude", bytes.NewReader(jsonBody))
	if err != nil {
		return nil, ArchitectConcludeOutput{}, NewInternalError("failed to create request: " + err.Error())
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(httpReq)
	if err != nil {
		return nil, ArchitectConcludeOutput{}, NewInternalError("failed to contact daemon: " + err.Error())
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, ArchitectConcludeOutput{}, wrapHTTPError(resp)
	}

	var result struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, ArchitectConcludeOutput{}, NewInternalError("failed to decode response: " + err.Error())
	}

	return nil, ArchitectConcludeOutput{
		Success: result.Success,
		Message: result.Message,
	}, nil
}

// wrapHTTPError converts an HTTP error response to a ToolError.
func wrapHTTPError(resp *http.Response) *ToolError {
	var errResp struct {
		Code  string `json:"code"`
		Error string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&errResp); err != nil {
		return NewInternalError(fmt.Sprintf("daemon returned status %d", resp.StatusCode))
	}
	msg := errResp.Error
	if msg == "" {
		msg = fmt.Sprintf("daemon returned status %d", resp.StatusCode)
	}

	switch resp.StatusCode {
	case http.StatusNotFound:
		return &ToolError{Code: ErrorCodeNotFound, Message: msg}
	case http.StatusBadRequest:
		return &ToolError{Code: ErrorCodeValidation, Message: msg}
	case http.StatusConflict:
		return &ToolError{Code: ErrorCodeStateConflict, Message: msg}
	default:
		return &ToolError{Code: ErrorCodeInternal, Message: msg}
	}
}

// joinParams joins query parameters with &.
func joinParams(params []string) string {
	result := ""
	for i, p := range params {
		if i > 0 {
			result += "&"
		}
		result += p
	}
	return result
}
