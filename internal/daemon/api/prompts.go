package api

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"

	projectconfig "github.com/kareemaly/cortex/internal/project/config"
	"github.com/kareemaly/cortex/internal/prompt"
	"github.com/kareemaly/cortex/internal/types"
)

// PromptHandlers provides HTTP handlers for prompt operations.
type PromptHandlers struct {
	deps *Dependencies
}

// NewPromptHandlers creates a new PromptHandlers with the given dependencies.
func NewPromptHandlers(deps *Dependencies) *PromptHandlers {
	return &PromptHandlers{deps: deps}
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

// List handles GET /prompts - lists all prompt files with ejection status.
func (h *PromptHandlers) List(w http.ResponseWriter, r *http.Request) {
	projectPath := GetProjectPath(r.Context())

	cfg, err := projectconfig.Load(projectPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "config_error", err.Error())
		return
	}

	extendPath := cfg.ResolvedExtendPath()
	if extendPath == "" {
		writeError(w, http.StatusBadRequest, "no_extend", "project has no 'extend' configured")
		return
	}

	baseDir := prompt.BasePromptsDir(extendPath)
	projectPromptsDir := prompt.PromptsDir(projectPath)

	// Walk base prompts directory to discover all .md files
	groupMap := make(map[string]*types.PromptGroupInfo)

	err = filepath.WalkDir(baseDir, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".md") {
			return nil
		}

		relPath, _ := filepath.Rel(baseDir, path)
		relPath = filepath.ToSlash(relPath)

		parts := strings.Split(relPath, "/")
		if len(parts) < 2 {
			return nil
		}

		group := parts[0]
		var subgroup string
		var stage string

		if len(parts) == 2 {
			stage = strings.TrimSuffix(parts[1], ".md")
		} else if len(parts) == 3 {
			subgroup = parts[1]
			stage = strings.TrimSuffix(parts[2], ".md")
		} else {
			return nil
		}

		// Check if ejected
		ejectedPath := filepath.Join(projectPromptsDir, relPath)
		_, ejectedErr := os.Stat(ejectedPath)
		ejected := ejectedErr == nil

		// Read content from ejected path if ejected, else from base
		var content string
		if ejected {
			data, readErr := os.ReadFile(ejectedPath)
			if readErr == nil {
				content = string(data)
			}
		} else {
			data, readErr := os.ReadFile(path)
			if readErr == nil {
				content = string(data)
			}
		}

		fileInfo := types.PromptFileInfo{
			Path:     relPath,
			Group:    group,
			Subgroup: subgroup,
			Stage:    stage,
			Ejected:  ejected,
			Content:  content,
		}

		// Build group key and display name
		var groupKey, groupName string
		if subgroup != "" {
			groupKey = group + "/" + subgroup
			groupName = strings.ToUpper(group) + " > " + strings.ToUpper(subgroup)
		} else {
			groupKey = group
			groupName = strings.ToUpper(group)
		}

		if _, ok := groupMap[groupKey]; !ok {
			groupMap[groupKey] = &types.PromptGroupInfo{
				Name: groupName,
				Key:  groupKey,
			}
		}
		groupMap[groupKey].Files = append(groupMap[groupKey].Files, fileInfo)

		return nil
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, "walk_error", fmt.Sprintf("failed to walk prompts directory: %s", err.Error()))
		return
	}

	// Sort groups by key
	var groups []types.PromptGroupInfo
	for _, g := range groupMap {
		sort.Slice(g.Files, func(i, j int) bool {
			return g.Files[i].Stage < g.Files[j].Stage
		})
		groups = append(groups, *g)
	}
	sort.Slice(groups, func(i, j int) bool {
		return groups[i].Key < groups[j].Key
	})

	// Read cortex.yaml content
	configPath := filepath.Join(projectPath, ".cortex", "cortex.yaml")
	configContent := ""
	if data, readErr := os.ReadFile(configPath); readErr == nil {
		configContent = string(data)
	}

	resp := ListPromptsResponse{
		Groups:        groups,
		ConfigPath:    configPath,
		ConfigContent: configContent,
	}
	writeJSON(w, http.StatusOK, resp)
}

// Eject handles POST /prompts/eject - copies a prompt from base to project for customization.
func (h *PromptHandlers) Eject(w http.ResponseWriter, r *http.Request) {
	projectPath := GetProjectPath(r.Context())

	var req EjectPromptRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid JSON in request body")
		return
	}

	if req.Path == "" {
		writeError(w, http.StatusBadRequest, "validation_error", "path is required")
		return
	}

	promptPath := strings.TrimPrefix(req.Path, "/")
	promptPath = filepath.Clean(promptPath)

	cfg, err := projectconfig.Load(projectPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "config_error", err.Error())
		return
	}

	extendPath := cfg.ResolvedExtendPath()
	if extendPath == "" {
		writeError(w, http.StatusBadRequest, "no_extend", "project has no 'extend' configured")
		return
	}

	sourcePath := filepath.Join(prompt.BasePromptsDir(extendPath), promptPath)
	destPath := filepath.Join(prompt.PromptsDir(projectPath), promptPath)

	// Validate source exists
	sourceInfo, err := os.Stat(sourcePath)
	if err != nil {
		if os.IsNotExist(err) {
			writeError(w, http.StatusNotFound, "not_found", fmt.Sprintf("source prompt not found: %s", promptPath))
			return
		}
		writeError(w, http.StatusInternalServerError, "stat_error", err.Error())
		return
	}
	if sourceInfo.IsDir() {
		writeError(w, http.StatusBadRequest, "validation_error", "cannot eject a directory")
		return
	}

	// Create destination directory
	destDir := filepath.Dir(destPath)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		writeError(w, http.StatusInternalServerError, "mkdir_error", err.Error())
		return
	}

	// Copy file
	if err := copyPromptFile(sourcePath, destPath); err != nil {
		writeError(w, http.StatusInternalServerError, "copy_error", err.Error())
		return
	}

	// Read the ejected content
	content, _ := os.ReadFile(destPath)

	// Parse path components
	parts := strings.Split(filepath.ToSlash(promptPath), "/")
	var group, subgroup, stage string
	if len(parts) == 2 {
		group = parts[0]
		stage = strings.TrimSuffix(parts[1], ".md")
	} else if len(parts) >= 3 {
		group = parts[0]
		subgroup = parts[1]
		stage = strings.TrimSuffix(parts[2], ".md")
	}

	resp := PromptFileInfo{
		Path:     filepath.ToSlash(promptPath),
		Group:    group,
		Subgroup: subgroup,
		Stage:    stage,
		Ejected:  true,
		Content:  string(content),
	}
	writeJSON(w, http.StatusOK, resp)
}

// Edit handles POST /prompts/edit - opens an ejected prompt in $EDITOR via tmux popup.
func (h *PromptHandlers) Edit(w http.ResponseWriter, r *http.Request) {
	projectPath := GetProjectPath(r.Context())

	var req EditPromptRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid JSON in request body")
		return
	}

	if req.Path == "" {
		writeError(w, http.StatusBadRequest, "validation_error", "path is required")
		return
	}

	promptPath := strings.TrimPrefix(req.Path, "/")
	promptPath = filepath.Clean(promptPath)

	filePath := filepath.Join(prompt.PromptsDir(projectPath), promptPath)

	// Validate file exists (must be ejected)
	if _, err := os.Stat(filePath); err != nil {
		if os.IsNotExist(err) {
			writeError(w, http.StatusNotFound, "not_found", "prompt file not found (not ejected?)")
			return
		}
		writeError(w, http.StatusInternalServerError, "stat_error", err.Error())
		return
	}

	if h.deps.TmuxManager == nil {
		writeError(w, http.StatusServiceUnavailable, "tmux_unavailable", "tmux is not installed")
		return
	}

	editor := os.Getenv("EDITOR")
	if editor == "" {
		editor = "vi"
	}

	command := fmt.Sprintf("%s %q", editor, filePath)

	projectCfg, cfgErr := projectconfig.Load(projectPath)
	tmuxSession := "cortex"
	if cfgErr == nil && projectCfg.Name != "" {
		tmuxSession = projectCfg.Name
	}

	if err := h.deps.TmuxManager.DisplayPopup(tmuxSession, "", command); err != nil {
		writeError(w, http.StatusInternalServerError, "tmux_error", fmt.Sprintf("failed to display popup: %s", err.Error()))
		return
	}

	writeJSON(w, http.StatusOK, ExecuteActionResponse{
		Success: true,
		Message: "Opened in editor",
	})
}

// Reset handles POST /prompts/reset - deletes an ejected prompt so it falls back to the built-in default.
func (h *PromptHandlers) Reset(w http.ResponseWriter, r *http.Request) {
	projectPath := GetProjectPath(r.Context())

	var req ResetPromptRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid JSON in request body")
		return
	}

	if req.Path == "" {
		writeError(w, http.StatusBadRequest, "validation_error", "path is required")
		return
	}

	promptPath := strings.TrimPrefix(req.Path, "/")
	promptPath = filepath.Clean(promptPath)

	projectPromptsDir := prompt.PromptsDir(projectPath)
	ejectedPath := filepath.Join(projectPromptsDir, promptPath)

	// Verify the file exists (i.e. is ejected)
	if _, err := os.Stat(ejectedPath); err != nil {
		if os.IsNotExist(err) {
			writeError(w, http.StatusBadRequest, "not_ejected", "prompt is not ejected")
			return
		}
		writeError(w, http.StatusInternalServerError, "stat_error", err.Error())
		return
	}

	if err := os.Remove(ejectedPath); err != nil {
		writeError(w, http.StatusInternalServerError, "remove_error", err.Error())
		return
	}

	// Clean up empty parent directories up to the prompts root
	removeEmptyParents(filepath.Dir(ejectedPath), projectPromptsDir)

	writeJSON(w, http.StatusOK, ExecuteActionResponse{
		Success: true,
		Message: "Prompt reset to default",
	})
}

// removeEmptyParents removes empty directories from dir up to (but not including) root.
func removeEmptyParents(dir, root string) {
	for dir != root && dir != "." && dir != "/" {
		entries, err := os.ReadDir(dir)
		if err != nil || len(entries) > 0 {
			return
		}
		if err := os.Remove(dir); err != nil {
			return
		}
		dir = filepath.Dir(dir)
	}
}

// copyPromptFile copies a file from src to dst.
func copyPromptFile(src, dst string) (err error) {
	sourceFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := sourceFile.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	destFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := destFile.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}()

	_, err = io.Copy(destFile, sourceFile)
	return err
}
