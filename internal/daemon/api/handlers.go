package api

import (
	"encoding/json"
	"net/http"

	"github.com/kareemaly/cortex/internal/tmux"
	"github.com/kareemaly/cortex/pkg/version"
)

// DaemonFocusHandler returns a handler that focuses the CortexDaemon dashboard window.
func DaemonFocusHandler(tmuxMgr *tmux.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if tmuxMgr == nil {
			writeError(w, http.StatusServiceUnavailable, "tmux_unavailable", "tmux is not installed")
			return
		}

		if err := tmuxMgr.FocusWindowByIndex("CortexDaemon", 0); err != nil {
			writeError(w, http.StatusInternalServerError, "focus_error", err.Error())
			return
		}
		if err := tmuxMgr.SwitchClient("CortexDaemon"); err != nil {
			writeError(w, http.StatusInternalServerError, "focus_error", err.Error())
			return
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(FocusResponse{Success: true, Window: "dashboard"})
	}
}

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
