package api

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"

	"github.com/kareemaly/cortex/internal/daemon/config"
	"github.com/kareemaly/cortex/internal/ticket"
)

// ProjectsListResponse is the response for GET /projects.
type ProjectsListResponse struct {
	Projects []ProjectResponse `json:"projects"`
}

// UnlinkProjectHandler returns a handler for DELETE /projects.
// It removes a project from the global registry without deleting any files.
func UnlinkProjectHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		projectPath := r.URL.Query().Get("path")
		if projectPath == "" {
			writeError(w, http.StatusBadRequest, "bad_request", "missing path parameter")
			return
		}

		cfg, err := config.Load()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to load config")
			return
		}

		if !cfg.UnregisterProject(projectPath) {
			writeError(w, http.StatusNotFound, "not_found", "project not found")
			return
		}

		if err := cfg.Save(); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to save config")
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// RegisterProjectRequest is the request body for POST /projects.
type RegisterProjectRequest struct {
	Path  string `json:"path"`
	Title string `json:"title,omitempty"`
}

// RegisterProjectHandler returns a handler for POST /projects.
func RegisterProjectHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req RegisterProjectRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, "invalid_json", "invalid JSON in request body")
			return
		}

		if req.Path == "" {
			writeError(w, http.StatusBadRequest, "validation_error", "path is required")
			return
		}

		// Resolve absolute path
		absPath := req.Path
		if !filepath.IsAbs(absPath) {
			writeError(w, http.StatusBadRequest, "validation_error", "path must be absolute")
			return
		}

		// Check .cortex directory exists
		cortexDir := filepath.Join(absPath, ".cortex")
		if _, err := os.Stat(cortexDir); err != nil {
			writeError(w, http.StatusBadRequest, "validation_error", "not a cortex project (no .cortex directory)")
			return
		}

		cfg, err := config.Load()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to load config")
			return
		}

		title := req.Title
		if title == "" {
			title = filepath.Base(absPath)
		}

		if !cfg.RegisterProject(absPath, title) {
			writeJSON(w, http.StatusOK, map[string]any{
				"success": true,
				"message": "project already registered",
			})
			return
		}

		if err := cfg.Save(); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to save config")
			return
		}

		writeJSON(w, http.StatusCreated, map[string]any{
			"success": true,
			"message": "project registered",
			"path":    absPath,
			"title":   title,
		})
	}
}

// ProjectsHandler returns a handler for GET /projects.
func ProjectsHandler(storeManager *StoreManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cfg, err := config.Load()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to load config")
			return
		}

		projects := make([]ProjectResponse, 0, len(cfg.Projects))
		for _, entry := range cfg.Projects {
			proj := ProjectResponse{
				Path:  entry.Path,
				Title: entry.Title,
			}

			// Check if the .cortex directory exists
			cortexDir := filepath.Join(entry.Path, ".cortex")
			if _, err := os.Stat(cortexDir); err == nil {
				proj.Exists = true

				// Best-effort ticket counts
				store, err := storeManager.GetStore(entry.Path)
				if err == nil {
					allTickets, err := store.ListAll()
					if err == nil {
						proj.Counts = &ProjectTicketCounts{
							Backlog:  len(allTickets[ticket.StatusBacklog]),
							Progress: len(allTickets[ticket.StatusProgress]),
							Review:   len(allTickets[ticket.StatusReview]),
							Done:     len(allTickets[ticket.StatusDone]),
						}
					}
				}
			}

			projects = append(projects, proj)
		}

		resp := ProjectsListResponse{Projects: projects}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to encode response")
		}
	}
}
