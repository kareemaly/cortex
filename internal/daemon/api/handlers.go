package api

import (
	"encoding/json"
	"net/http"

	"github.com/kareemaly/cortex/pkg/version"
)

// HealthHandler returns the health check handler.
func HealthHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		resp := HealthResponse{
			Status:  "ok",
			Version: version.Version,
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			writeError(w, http.StatusInternalServerError, "internal_error", "failed to encode response")
		}
	}
}
