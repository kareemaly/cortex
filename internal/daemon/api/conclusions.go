package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/kareemaly/cortex/internal/conclusion"
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

// parseIntQuery parses an integer query parameter, returning defaultVal on missing or invalid input.
func parseIntQuery(r *http.Request, key string, defaultVal int) int {
	s := r.URL.Query().Get(key)
	if s == "" {
		return defaultVal
	}
	v, err := strconv.Atoi(s)
	if err != nil || v < 0 {
		return defaultVal
	}
	return v
}

// List handles GET /conclusions - lists conclusions with optional type filter and pagination.
func (h *ConclusionHandlers) List(w http.ResponseWriter, r *http.Request) {
	projectPath := GetProjectPath(r.Context())

	if h.deps.ConclusionStoreManager == nil {
		writeJSON(w, http.StatusOK, types.ListConclusionsResponse{Conclusions: []types.ConclusionSummary{}, Total: 0})
		return
	}

	store, err := h.deps.ConclusionStoreManager.GetStore(projectPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "store_error", err.Error())
		return
	}

	opts := conclusion.ListOptions{
		Type:   r.URL.Query().Get("type"),
		Limit:  parseIntQuery(r, "limit", 0),
		Offset: parseIntQuery(r, "offset", 0),
	}

	conclusions, total, err := store.ListWithOptions(opts)
	if err != nil {
		handleConclusionError(w, err, h.deps.Logger)
		return
	}

	summaries := make([]types.ConclusionSummary, len(conclusions))
	for i, c := range conclusions {
		summaries[i] = types.ConclusionSummary{
			ID:      c.ID,
			Type:    string(c.Type),
			Ticket:  c.Ticket,
			Repo:    c.Repo,
			Created: c.Created,
		}
	}

	writeJSON(w, http.StatusOK, types.ListConclusionsResponse{Conclusions: summaries, Total: total})
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
