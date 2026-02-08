package api

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
)

// EventHandlers handles SSE event streaming.
type EventHandlers struct {
	deps *Dependencies
}

// NewEventHandlers creates a new EventHandlers.
func NewEventHandlers(deps *Dependencies) *EventHandlers {
	return &EventHandlers{deps: deps}
}

// Stream handles GET /events and streams SSE events for the project.
func (h *EventHandlers) Stream(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, "internal_error", "streaming not supported")
		return
	}

	projectPath := GetProjectPath(r.Context())

	ch, unsubscribe := h.deps.Bus.Subscribe(projectPath)
	defer unsubscribe()

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	for {
		select {
		case <-r.Context().Done():
			return
		case event, ok := <-ch:
			if !ok {
				return
			}
			data, err := json.Marshal(event)
			if err != nil {
				slog.Warn("failed to marshal SSE event", "error", err)
				continue
			}
			if _, err := fmt.Fprintf(w, "data: %s\n\n", data); err != nil {
				slog.Warn("failed to write SSE event", "error", err)
				return
			}
			flusher.Flush()
		}
	}
}
