package api

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"

	"github.com/kareemaly/cortex/internal/daemon/config"
	"github.com/kareemaly/cortex/internal/ticket"
)

// ProjectTicketCounts holds ticket counts by status.
type ProjectTicketCounts struct {
	Backlog  int `json:"backlog"`
	Progress int `json:"progress"`
	Review   int `json:"review"`
	Done     int `json:"done"`
}

// ProjectResponse represents a single project in the API response.
type ProjectResponse struct {
	Path   string               `json:"path"`
	Title  string               `json:"title"`
	Exists bool                 `json:"exists"`
	Counts *ProjectTicketCounts `json:"counts,omitempty"`
}

// ProjectsListResponse is the response for GET /projects.
type ProjectsListResponse struct {
	Projects []ProjectResponse `json:"projects"`
}

// ProjectsHandler returns a handler for GET /projects.
func ProjectsHandler(storeManager *StoreManager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cfg, err := config.Load()
		if err != nil {
			http.Error(w, "failed to load config", http.StatusInternalServerError)
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
			http.Error(w, "failed to encode response", http.StatusInternalServerError)
		}
	}
}
