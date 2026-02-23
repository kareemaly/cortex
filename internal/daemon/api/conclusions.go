package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/kareemaly/cortex/internal/types"
)

// ConclusionHandlers provides HTTP handlers for conclusion operations.
type ConclusionHandlers struct {
	deps *Dependencies
}

// NewConclusionHandlers creates a new ConclusionHandlers with the given dependencies.
func NewConclusionHandlers(deps *Dependencies) *ConclusionHandlers {
	return &ConclusionHandlers{deps: deps}
}

// List handles GET /conclusions - lists all conclusions.
func (h *ConclusionHandlers) List(w http.ResponseWriter, r *http.Request) {
	projectPath := GetProjectPath(r.Context())

	if h.deps.ConclusionStoreManager == nil {
		writeJSON(w, http.StatusOK, types.ListConclusionsResponse{Conclusions: []types.ConclusionResponse{}})
		return
	}

	store, err := h.deps.ConclusionStoreManager.GetStore(projectPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "store_error", err.Error())
		return
	}

	conclusions, err := store.List()
	if err != nil {
		handleConclusionError(w, err, h.deps.Logger)
		return
	}

	resp := make([]types.ConclusionResponse, len(conclusions))
	for i, c := range conclusions {
		resp[i] = types.ConclusionResponse{
			ID:      c.ID,
			Type:    string(c.Type),
			Ticket:  c.Ticket,
			Repo:    c.Repo,
			Body:    c.Body,
			Created: c.Created,
		}
	}

	writeJSON(w, http.StatusOK, types.ListConclusionsResponse{Conclusions: resp})
}

// Get handles GET /conclusions/{id} - gets a conclusion by ID.
func (h *ConclusionHandlers) Get(w http.ResponseWriter, r *http.Request) {
	projectPath := GetProjectPath(r.Context())

	if h.deps.ConclusionStoreManager == nil {
		writeError(w, http.StatusNotFound, "not_found", "conclusion store not available")
		return
	}

	store, err := h.deps.ConclusionStoreManager.GetStore(projectPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "store_error", err.Error())
		return
	}

	id := chi.URLParam(r, "id")
	c, err := store.Get(id)
	if err != nil {
		handleConclusionError(w, err, h.deps.Logger)
		return
	}

	resp := types.ConclusionResponse{
		ID:      c.ID,
		Type:    string(c.Type),
		Ticket:  c.Ticket,
		Repo:    c.Repo,
		Body:    c.Body,
		Created: c.Created,
	}

	writeJSON(w, http.StatusOK, resp)
}

// Create handles POST /conclusions - creates a new conclusion.
func (h *ConclusionHandlers) Create(w http.ResponseWriter, r *http.Request) {
	projectPath := GetProjectPath(r.Context())

	if h.deps.ConclusionStoreManager == nil {
		writeError(w, http.StatusInternalServerError, "store_error", "conclusion store not available")
		return
	}

	store, err := h.deps.ConclusionStoreManager.GetStore(projectPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "store_error", err.Error())
		return
	}

	var req CreateConclusionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid JSON in request body")
		return
	}

	if req.Body == "" {
		writeError(w, http.StatusBadRequest, "validation_error", "body cannot be empty")
		return
	}

	c, err := store.Create(req.Type, req.Ticket, req.Repo, req.Body)
	if err != nil {
		handleConclusionError(w, err, h.deps.Logger)
		return
	}

	resp := types.ConclusionResponse{
		ID:      c.ID,
		Type:    string(c.Type),
		Ticket:  c.Ticket,
		Repo:    c.Repo,
		Body:    c.Body,
		Created: c.Created,
	}

	writeJSON(w, http.StatusCreated, resp)
}
