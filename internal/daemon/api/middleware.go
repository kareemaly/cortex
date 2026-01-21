package api

import (
	"context"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

// ProjectHeader is the HTTP header name for specifying the project path.
const ProjectHeader = "X-Cortex-Project"

type contextKey string

const projectPathKey contextKey = "projectPath"

// ProjectRequired returns middleware that requires the X-Cortex-Project header.
// It validates that:
// 1. The header is present
// 2. The path is absolute
// 3. The path exists and has a .cortex/tickets directory
func ProjectRequired() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			projectPath := r.Header.Get(ProjectHeader)

			// Check header is present
			if projectPath == "" {
				writeError(w, http.StatusBadRequest, "missing_project_header", "X-Cortex-Project header required")
				return
			}

			// Check path is absolute
			if !filepath.IsAbs(projectPath) {
				writeError(w, http.StatusBadRequest, "invalid_project_path", "project path must be absolute")
				return
			}

			// Check path exists
			if _, err := os.Stat(projectPath); os.IsNotExist(err) {
				writeError(w, http.StatusNotFound, "project_not_found", "project path does not exist")
				return
			}

			// Check .cortex/tickets directory exists
			ticketsDir := filepath.Join(projectPath, ".cortex", "tickets")
			if _, err := os.Stat(ticketsDir); os.IsNotExist(err) {
				writeError(w, http.StatusNotFound, "project_not_found", "not a cortex project (no .cortex/tickets directory)")
				return
			}

			// Add project path to context
			ctx := context.WithValue(r.Context(), projectPathKey, projectPath)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetProjectPath returns the project path from the request context.
func GetProjectPath(ctx context.Context) string {
	if v := ctx.Value(projectPathKey); v != nil {
		return v.(string)
	}
	return ""
}

// RequestLogger returns a middleware that logs HTTP requests.
func RequestLogger(logger *slog.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			next.ServeHTTP(ww, r)

			logger.Info("request",
				"method", r.Method,
				"path", r.URL.Path,
				"status", ww.Status(),
				"bytes", ww.BytesWritten(),
				"duration_ms", time.Since(start).Milliseconds(),
				"request_id", middleware.GetReqID(r.Context()),
			)
		})
	}
}
