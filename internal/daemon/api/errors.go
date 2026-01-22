package api

import (
	"encoding/json"
	"net/http"

	"github.com/kareemaly/cortex/internal/ticket"
)

// writeJSON writes a JSON response with the given status code.
func writeJSON(w http.ResponseWriter, status int, data any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

// writeError writes an error response with the given status code.
func writeError(w http.ResponseWriter, status int, code, message string) {
	writeJSON(w, status, ErrorResponse{
		Error: message,
		Code:  code,
	})
}

// handleTicketError converts ticket store errors to HTTP responses.
func handleTicketError(w http.ResponseWriter, err error) {
	switch e := err.(type) {
	case *ticket.NotFoundError:
		writeError(w, http.StatusNotFound, "not_found", e.Error())
	case *ticket.ValidationError:
		writeError(w, http.StatusBadRequest, "validation_error", e.Error())
	default:
		writeError(w, http.StatusInternalServerError, "internal_error", "internal server error")
	}
}

// validStatus returns true if the status is valid.
func validStatus(status string) bool {
	switch ticket.Status(status) {
	case ticket.StatusBacklog, ticket.StatusProgress, ticket.StatusReview, ticket.StatusDone:
		return true
	default:
		return false
	}
}
