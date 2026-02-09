package api

import (
	"encoding/json"
	"net/http"
	"os"

	daemonconfig "github.com/kareemaly/cortex/internal/daemon/config"
	projectconfig "github.com/kareemaly/cortex/internal/project/config"
	"gopkg.in/yaml.v3"
)

// ConfigHandlers provides HTTP handlers for configuration operations.
type ConfigHandlers struct {
	deps *Dependencies
}

// NewConfigHandlers creates a new ConfigHandlers with the given dependencies.
func NewConfigHandlers(deps *Dependencies) *ConfigHandlers {
	return &ConfigHandlers{deps: deps}
}

// ReadProjectConfigResponse is the response for GET /config/project.
type ReadProjectConfigResponse struct {
	Content string `json:"content"`
	Path    string `json:"path"`
}

// UpdateProjectConfigRequest is the request for PUT /config/project.
type UpdateProjectConfigRequest struct {
	Content string `json:"content"`
}

// ReadProjectConfig handles GET /config/project - reads project's cortex.yaml.
func (h *ConfigHandlers) ReadProjectConfig(w http.ResponseWriter, r *http.Request) {
	projectPath := GetProjectPath(r.Context())

	configPath := projectPath + "/.cortex/cortex.yaml"
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			writeError(w, http.StatusNotFound, "not_found", "project config not found")
			return
		}
		writeError(w, http.StatusInternalServerError, "read_error", "failed to read project config")
		return
	}

	writeJSON(w, http.StatusOK, ReadProjectConfigResponse{
		Content: string(data),
		Path:    configPath,
	})
}

// UpdateProjectConfig handles PUT /config/project - updates project's cortex.yaml.
func (h *ConfigHandlers) UpdateProjectConfig(w http.ResponseWriter, r *http.Request) {
	projectPath := GetProjectPath(r.Context())

	var req UpdateProjectConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid JSON in request body")
		return
	}

	if req.Content == "" {
		writeError(w, http.StatusBadRequest, "validation_error", "content cannot be empty")
		return
	}

	// Validate the YAML is parseable as a project config
	var testCfg projectconfig.Config
	if err := yaml.Unmarshal([]byte(req.Content), &testCfg); err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", "invalid YAML: "+err.Error())
		return
	}

	configPath := projectPath + "/.cortex/cortex.yaml"
	if err := os.WriteFile(configPath, []byte(req.Content), 0644); err != nil {
		writeError(w, http.StatusInternalServerError, "write_error", "failed to write project config")
		return
	}

	writeJSON(w, http.StatusOK, ReadProjectConfigResponse{
		Content: req.Content,
		Path:    configPath,
	})
}

// ReadGlobalConfigResponse is the response for GET /config/global.
type ReadGlobalConfigResponse struct {
	Content string `json:"content"`
	Path    string `json:"path"`
}

// UpdateGlobalConfigRequest is the request for PUT /config/global.
type UpdateGlobalConfigRequest struct {
	Content string `json:"content"`
}

// ReadGlobalConfig handles GET /config/global - reads ~/.cortex/settings.yaml.
func (h *ConfigHandlers) ReadGlobalConfig(w http.ResponseWriter, r *http.Request) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to get home directory")
		return
	}

	configPath := homeDir + "/.cortex/settings.yaml"
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// Return defaults if no file exists
			cfg := daemonconfig.DefaultConfig()
			content, _ := yaml.Marshal(cfg)
			writeJSON(w, http.StatusOK, ReadGlobalConfigResponse{
				Content: string(content),
				Path:    configPath,
			})
			return
		}
		writeError(w, http.StatusInternalServerError, "read_error", "failed to read global config")
		return
	}

	writeJSON(w, http.StatusOK, ReadGlobalConfigResponse{
		Content: string(data),
		Path:    configPath,
	})
}

// UpdateGlobalConfig handles PUT /config/global - updates ~/.cortex/settings.yaml.
func (h *ConfigHandlers) UpdateGlobalConfig(w http.ResponseWriter, r *http.Request) {
	var req UpdateGlobalConfigRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid JSON in request body")
		return
	}

	if req.Content == "" {
		writeError(w, http.StatusBadRequest, "validation_error", "content cannot be empty")
		return
	}

	// Validate the YAML is parseable as daemon config
	var testCfg daemonconfig.Config
	if err := yaml.Unmarshal([]byte(req.Content), &testCfg); err != nil {
		writeError(w, http.StatusBadRequest, "validation_error", "invalid YAML: "+err.Error())
		return
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal_error", "failed to get home directory")
		return
	}

	configPath := homeDir + "/.cortex/settings.yaml"
	if err := os.WriteFile(configPath, []byte(req.Content), 0600); err != nil {
		writeError(w, http.StatusInternalServerError, "write_error", "failed to write global config")
		return
	}

	writeJSON(w, http.StatusOK, ReadGlobalConfigResponse{
		Content: req.Content,
		Path:    configPath,
	})
}
