package api

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/kareemaly/cortex/internal/types"
)

// NoteHandlers provides HTTP handlers for note operations.
type NoteHandlers struct {
	deps *Dependencies
}

// NewNoteHandlers creates a new NoteHandlers with the given dependencies.
func NewNoteHandlers(deps *Dependencies) *NoteHandlers {
	return &NoteHandlers{deps: deps}
}

// List handles GET /notes - lists all notes.
func (h *NoteHandlers) List(w http.ResponseWriter, r *http.Request) {
	projectPath := GetProjectPath(r.Context())
	store, err := h.deps.NotesStoreManager.GetStore(projectPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "store_error", err.Error())
		return
	}

	notesList, err := store.List()
	if err != nil {
		handleNoteError(w, err, h.deps.Logger)
		return
	}

	resp := make([]types.NoteResponse, len(notesList))
	for i, n := range notesList {
		resp[i] = types.NoteResponse{
			ID:      n.ID,
			Text:    n.Text,
			Due:     n.Due,
			Created: n.Created,
		}
	}

	writeJSON(w, http.StatusOK, ListNotesResponse{Notes: resp})
}

// Create handles POST /notes - creates a new note.
func (h *NoteHandlers) Create(w http.ResponseWriter, r *http.Request) {
	var req CreateNoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid JSON in request body")
		return
	}

	projectPath := GetProjectPath(r.Context())
	store, err := h.deps.NotesStoreManager.GetStore(projectPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "store_error", err.Error())
		return
	}

	var due *time.Time
	if req.Due != nil && *req.Due != "" {
		parsed, parseErr := time.Parse(time.DateOnly, *req.Due)
		if parseErr != nil {
			writeError(w, http.StatusBadRequest, "validation_error", "due must be YYYY-MM-DD format")
			return
		}
		due = &parsed
	}

	note, err := store.Create(req.Text, due)
	if err != nil {
		handleNoteError(w, err, h.deps.Logger)
		return
	}

	resp := types.NoteResponse{
		ID:      note.ID,
		Text:    note.Text,
		Due:     note.Due,
		Created: note.Created,
	}
	writeJSON(w, http.StatusCreated, resp)
}

// Update handles PUT /notes/{id} - updates a note.
func (h *NoteHandlers) Update(w http.ResponseWriter, r *http.Request) {
	projectPath := GetProjectPath(r.Context())
	store, err := h.deps.NotesStoreManager.GetStore(projectPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "store_error", err.Error())
		return
	}

	id := chi.URLParam(r, "id")

	var req UpdateNoteRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid_json", "invalid JSON in request body")
		return
	}

	note, err := store.Update(id, req.Text, req.Due)
	if err != nil {
		handleNoteError(w, err, h.deps.Logger)
		return
	}

	resp := types.NoteResponse{
		ID:      note.ID,
		Text:    note.Text,
		Due:     note.Due,
		Created: note.Created,
	}
	writeJSON(w, http.StatusOK, resp)
}

// Delete handles DELETE /notes/{id} - deletes a note.
func (h *NoteHandlers) Delete(w http.ResponseWriter, r *http.Request) {
	projectPath := GetProjectPath(r.Context())
	store, err := h.deps.NotesStoreManager.GetStore(projectPath)
	if err != nil {
		writeError(w, http.StatusInternalServerError, "store_error", err.Error())
		return
	}

	id := chi.URLParam(r, "id")
	if err := store.Delete(id); err != nil {
		handleNoteError(w, err, h.deps.Logger)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}
