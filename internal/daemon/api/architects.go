package api

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"

	"github.com/kareemaly/cortex/internal/daemon/config"
	"github.com/kareemaly/cortex/internal/ticket"
)

// ArchitectsListResponse is the response for GET /architects.
type ArchitectsListResponse struct {
	Architects []ArchitectResponse `json:"architects"`
}

// UnlinkArchitectHandler returns a handler for DELETE /architects.
// It removes an architect from the global registry without deleting any files.
func UnlinkArchitectHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		architectPath := r.URL.Query().Get("path")
		if architectPath == "" {
			writeError(w, http.StatusBadRequest, "bad_request", "missing path parameter")
			return
		}

		cfg, err := config.Load()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to load config")
			return
		}

		if !cfg.UnregisterArchitect(architectPath) {
			writeError(w, http.StatusNotFound, "not_found", "architect not found")
			return
		}

		if err := cfg.Save(); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to save config")
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}

// RegisterArchitectRequest is the request body for POST /architects.
type RegisterArchitectRequest struct {
	Path  string `json:"path"`
	Title string `json:"title,omitempty"`
}

// RegisterArchitectHandler returns a handler for POST /architects.
func RegisterArchitectHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var req RegisterArchitectRequest
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

		// Check for cortex.yaml (architect marker)
		cortexYaml := filepath.Join(absPath, "cortex.yaml")
		if _, err := os.Stat(cortexYaml); os.IsNotExist(err) {
			writeError(w, http.StatusBadRequest, "validation_error", "not a cortex architect (no cortex.yaml)")
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

		if !cfg.RegisterArchitect(absPath, title) {
			writeJSON(w, http.StatusOK, map[string]any{
				"success": true,
				"message": "architect already registered",
			})
			return
		}

		if err := cfg.Save(); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to save config")
			return
		}

		writeJSON(w, http.StatusCreated, map[string]any{
			"success": true,
			"message": "architect registered",
			"path":    absPath,
			"title":   title,
		})
	}
}

// ArchitectsHandler returns a handler for GET /architects.
func ArchitectsHandler(storeManager *StoreManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cfg, err := config.Load()
		if err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to load config")
			return
		}

		architects := make([]ArchitectResponse, 0, len(cfg.Architects))
		for _, entry := range cfg.Architects {
			arch := ArchitectResponse{
				Path:  entry.Path,
				Title: entry.Title,
			}

			// Check if architect exists (cortex.yaml)
			cortexYaml := filepath.Join(entry.Path, "cortex.yaml")
			if _, err := os.Stat(cortexYaml); err == nil {
				arch.Exists = true

				// Best-effort ticket counts
				store, err := storeManager.GetStore(entry.Path)
				if err == nil {
					allTickets, err := store.ListAll()
					if err == nil {
						arch.Counts = &ArchitectTicketCounts{
							Backlog:  len(allTickets[ticket.StatusBacklog]),
							Progress: len(allTickets[ticket.StatusProgress]),
							Done:     len(allTickets[ticket.StatusDone]),
						}
					}
				}
			}

			architects = append(architects, arch)
		}

		resp := ArchitectsListResponse{Architects: architects}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to encode response")
		}
	}
}
