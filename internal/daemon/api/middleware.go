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

// ArchitectHeader is the HTTP header name for specifying the architect path.
const ArchitectHeader = "X-Cortex-Architect"

type contextKey string

const architectPathKey contextKey = "architectPath"

// ArchitectRequired returns middleware that requires the X-Cortex-Architect header.
// It validates that:
// 1. The header is present
// 2. The path is absolute
// 3. The path exists and has a cortex.yaml file
func ArchitectRequired() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			architectPath := r.Header.Get(ArchitectHeader)

			// Check header is present
			if architectPath == "" {
				writeError(w, http.StatusBadRequest, "missing_architect_header", "X-Cortex-Architect header required")
				return
			}

			// Check path is absolute
			if !filepath.IsAbs(architectPath) {
				writeError(w, http.StatusBadRequest, "invalid_architect_path", "architect path must be absolute")
				return
			}

			// Check path exists
			if _, err := os.Stat(architectPath); os.IsNotExist(err) {
				writeError(w, http.StatusNotFound, "architect_not_found", "architect path does not exist")
				return
			}

			// Check for cortex.yaml (architect marker)
			cortexYaml := filepath.Join(architectPath, "cortex.yaml")
			if _, err := os.Stat(cortexYaml); os.IsNotExist(err) {
				writeError(w, http.StatusNotFound, "architect_not_found", "not a cortex architect (no cortex.yaml)")
				return
			}

			// Add architect path to context
			ctx := context.WithValue(r.Context(), architectPathKey, filepath.Clean(architectPath))
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetArchitectPath returns the architect path from the request context.
func GetArchitectPath(ctx context.Context) string {
	if v := ctx.Value(architectPathKey); v != nil {
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
