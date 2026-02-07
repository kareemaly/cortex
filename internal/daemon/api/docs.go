package api

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/kareemaly/cortex/internal/storage"
	"github.com/kareemaly/cortex/internal/types"
)

// DocHandlers provides HTTP handlers for doc operations.
type DocHandlers struct {
	deps *Dependencies
}

// NewDocHandlers creates a new DocHandlers with the given dependencies.
func NewDocHandlers(deps *Dependencies) *DocHandlers {
	return &DocHandlers{deps: deps}
}

// List handles GET /docs - lists docs with optional filters.
func (h *DocHandlers) List(w http.ResponseWriter, r *http.Request) {
	projectPath := GetProjectPath(r.Context())
	store, err := h.deps.DocsStoreManager.GetStore(projectPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "store_error", err.Error())
		return
	}

	category := r.URL.Query().Get("category")
	tag := r.URL.Query().Get("tag")
	query := r.URL.Query().Get("query")

	docList, err := store.List(category, tag, query)
	if err != nil {
		handleDocError(w, err, h.deps.Logger)
		return
	}

	summaries := make([]DocSummary, len(docList))
	for i, d := range docList {
		if query != "" {
			summaries[i] = types.ToDocSummaryWithQuery(d, query)
		} else {
			summaries[i] = types.ToDocSummary(d)
		}
	}

	writeJSON(w, http.StatusOK, ListDocsResponse{Docs: summaries})
}

// Create handles POST /docs - creates a new doc.
func (h *DocHandlers) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateDocRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid JSON in request body")
		return
	}

	projectPath := GetProjectPath(r.Context())
	store, err := h.deps.DocsStoreManager.GetStore(projectPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "store_error", err.Error())
		return
	}

	d, err := store.Create(req.Title, req.Category, req.Body, req.Tags, req.References)
	if err != nil {
		handleDocError(w, err, h.deps.Logger)
		return
	}

	resp := types.ToDocResponse(d)
	writeJSON(w, http.StatusCreated, resp)
}

// Get handles GET /docs/{id} - gets a specific doc.
func (h *DocHandlers) Get(w http.ResponseWriter, r *http.Request) {
	projectPath := GetProjectPath(r.Context())
	store, err := h.deps.DocsStoreManager.GetStore(projectPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "store_error", err.Error())
		return
	}

	id := chi.URLParam(r, "id")
	d, err := store.Get(id)
	if err != nil {
		handleDocError(w, err, h.deps.Logger)
		return
	}

	resp := types.ToDocResponse(d)
	writeJSON(w, http.StatusOK, resp)
}

// Update handles PUT /docs/{id} - updates a doc.
func (h *DocHandlers) Update(w http.ResponseWriter, r *http.Request) {
	projectPath := GetProjectPath(r.Context())
	store, err := h.deps.DocsStoreManager.GetStore(projectPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "store_error", err.Error())
		return
	}

	id := chi.URLParam(r, "id")

	var req UpdateDocRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid JSON in request body")
		return
	}

	d, err := store.Update(id, req.Title, req.Body, req.Tags, req.References)
	if err != nil {
		handleDocError(w, err, h.deps.Logger)
		return
	}

	resp := types.ToDocResponse(d)
	writeJSON(w, http.StatusOK, resp)
}

// AddComment handles POST /docs/{id}/comments - adds a comment to a doc.
func (h *DocHandlers) AddComment(w http.ResponseWriter, r *http.Request) {
	projectPath := GetProjectPath(r.Context())
	store, err := h.deps.DocsStoreManager.GetStore(projectPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "store_error", err.Error())
		return
	}

	id := chi.URLParam(r, "id")
	_, err = store.Get(id)
	if err != nil {
		handleDocError(w, err, h.deps.Logger)
		return
	}

	var req AddDocCommentRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid JSON in request body")
		return
	}

	// Validate comment type
	commentType := storage.CommentType(req.Type)
	switch commentType {
	case storage.CommentReviewRequested, storage.CommentDone, storage.CommentBlocker,
		storage.CommentGeneral:
		// Valid type
	default:
		writeError(w, http.StatusBadRequest, "validation_error", "invalid comment type: must be review_requested, done, blocker, or comment")
		return
	}

	author := req.Author
	if author == "" {
		author = "unknown"
	}

	comment, err := store.AddComment(id, author, commentType, req.Content, nil)
	if err != nil {
		handleDocError(w, err, h.deps.Logger)
		return
	}

	resp := AddCommentResponse{
		Success: true,
		Comment: types.ToCommentResponse(comment),
	}
	writeJSON(w, http.StatusOK, resp)
}

// Delete handles DELETE /docs/{id} - deletes a doc.
func (h *DocHandlers) Delete(w http.ResponseWriter, r *http.Request) {
	projectPath := GetProjectPath(r.Context())
	store, err := h.deps.DocsStoreManager.GetStore(projectPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "store_error", err.Error())
		return
	}

	id := chi.URLParam(r, "id")
	if err := store.Delete(id); err != nil {
		handleDocError(w, err, h.deps.Logger)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Move handles POST /docs/{id}/move - moves a doc to a different category.
func (h *DocHandlers) Move(w http.ResponseWriter, r *http.Request) {
	projectPath := GetProjectPath(r.Context())
	store, err := h.deps.DocsStoreManager.GetStore(projectPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "store_error", err.Error())
		return
	}

	id := chi.URLParam(r, "id")

	var req MoveDocRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid JSON in request body")
		return
	}

	d, err := store.Move(id, req.Category)
	if err != nil {
		handleDocError(w, err, h.deps.Logger)
		return
	}

	resp := types.ToDocResponse(d)
	writeJSON(w, http.StatusOK, resp)
}
