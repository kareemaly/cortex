# Daemon Skeleton

Set up the cortexd HTTP server with routing, middleware, configuration, and logging.

## Context

The project foundation is complete. Now we need a running daemon that can serve HTTP requests.

Reference `~/projects/cortex/internal/daemon/` for patterns, but simplify per DESIGN.md (no SQLite, no git wrapper).

## Requirements

### 1. Dependencies

Add to go.mod:
```
github.com/go-chi/chi/v5
gopkg.in/yaml.v3
gopkg.in/natefinch/lumberjack.v2
```

### 2. cmd/cortexd/main.go

Expand to full daemon entry point:

```go
package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/kareemaly/cortex1/internal/daemon/api"
	"github.com/kareemaly/cortex1/internal/daemon/config"
	"github.com/kareemaly/cortex1/internal/daemon/logging"
	"github.com/kareemaly/cortex1/pkg/version"
)

func main() {
	// Load config
	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		os.Exit(1)
	}

	// Setup logging
	logger := logging.Setup(cfg.LogLevel)
	logger.Info("starting cortexd", "version", version.Version)

	// Create and start server
	srv := api.NewServer(cfg, logger)

	// Graceful shutdown
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		logger.Info("shutting down...")
		cancel()
	}()

	if err := srv.Run(ctx); err != nil {
		logger.Error("server error", "error", err)
		os.Exit(1)
	}
}
```

### 3. internal/daemon/config/config.go

Configuration loading from `~/.cortex/settings.yaml`:

```go
package config

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Port     int    `yaml:"port"`
	LogLevel string `yaml:"log_level"`
}

func DefaultConfig() *Config {
	return &Config{
		Port:     4200,
		LogLevel: "info",
	}
}

func Load() (*Config, error) {
	cfg := DefaultConfig()

	home, err := os.UserHomeDir()
	if err != nil {
		return cfg, nil // Use defaults if can't find home
	}

	path := filepath.Join(home, ".cortex", "settings.yaml")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil // Use defaults if no config file
		}
		return nil, err
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}
```

### 4. internal/daemon/logging/logging.go

Structured logging setup:

```go
package logging

import (
	"log/slog"
	"os"
	"path/filepath"

	"gopkg.in/natefinch/lumberjack.v2"
)

func Setup(level string) *slog.Logger {
	var logLevel slog.Level
	switch level {
	case "debug":
		logLevel = slog.LevelDebug
	case "warn":
		logLevel = slog.LevelWarn
	case "error":
		logLevel = slog.LevelError
	default:
		logLevel = slog.LevelInfo
	}

	// Log to file with rotation
	home, _ := os.UserHomeDir()
	logPath := filepath.Join(home, ".cortex", "daemon.log")

	// Ensure directory exists
	os.MkdirAll(filepath.Dir(logPath), 0755)

	writer := &lumberjack.Logger{
		Filename:   logPath,
		MaxSize:    10, // MB
		MaxBackups: 3,
		MaxAge:     7, // days
	}

	handler := slog.NewJSONHandler(writer, &slog.HandlerOptions{
		Level: logLevel,
	})

	return slog.New(handler)
}
```

### 5. internal/daemon/api/server.go

HTTP server with Chi router:

```go
package api

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/kareemaly/cortex1/internal/daemon/config"
)

type Server struct {
	cfg    *config.Config
	logger *slog.Logger
	router *chi.Mux
	server *http.Server
}

func NewServer(cfg *config.Config, logger *slog.Logger) *Server {
	r := chi.NewRouter()

	// Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(requestLogger(logger))

	s := &Server{
		cfg:    cfg,
		logger: logger,
		router: r,
	}

	s.routes()
	return s
}

func (s *Server) routes() {
	s.router.Get("/health", s.handleHealth)
}

func (s *Server) Run(ctx context.Context) error {
	s.server = &http.Server{
		Addr:         fmt.Sprintf(":%d", s.cfg.Port),
		Handler:      s.router,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 0, // Disabled for SSE
		IdleTimeout:  60 * time.Second,
	}

	s.logger.Info("server listening", "port", s.cfg.Port)

	errCh := make(chan error, 1)
	go func() {
		errCh <- s.server.ListenAndServe()
	}()

	select {
	case err := <-errCh:
		if err != http.ErrServerClosed {
			return err
		}
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		return s.server.Shutdown(shutdownCtx)
	}

	return nil
}
```

### 6. internal/daemon/api/middleware.go

Request logging middleware:

```go
package api

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5/middleware"
)

func requestLogger(logger *slog.Logger) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)

			defer func() {
				logger.Info("request",
					"method", r.Method,
					"path", r.URL.Path,
					"status", ww.Status(),
					"duration", time.Since(start),
					"request_id", middleware.GetReqID(r.Context()),
				)
			}()

			next.ServeHTTP(ww, r)
		})
	}
}
```

### 7. internal/daemon/api/handlers.go

Health check handler:

```go
package api

import (
	"encoding/json"
	"net/http"

	"github.com/kareemaly/cortex1/pkg/version"
)

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]any{
		"status":  "ok",
		"version": version.Version,
	})
}
```

### 8. Remove .gitkeep files

Delete these files as they're replaced by real code:
- `internal/daemon/api/.gitkeep`
- `internal/daemon/config/.gitkeep`

Keep `internal/daemon/mcp/.gitkeep` (MCP ticket will handle it).

## Verification

```bash
# Build succeeds
make build

# Daemon starts and listens
./bin/cortexd &
# Should see: "server listening" in logs

# Health check works
curl http://localhost:4200/health
# Should return: {"status":"ok","version":"dev"}

# Graceful shutdown works
kill %1
# Should see: "shutting down..." in logs

# Lint passes
make lint
```

## Notes

- WriteTimeout is 0 to support SSE later
- Config file is optional - daemon uses defaults if missing
- Logs go to ~/.cortex/daemon.log with rotation
- No business logic yet - just the HTTP skeleton

## Implementation

### Commits
- `9e3403e` feat: add daemon HTTP server skeleton with Chi router and graceful shutdown

### Key Files Changed
- `cmd/cortexd/main.go` - Full daemon entry point with signal handling
- `internal/daemon/config/config.go` - Config loading from ~/.cortex/settings.yaml
- `internal/daemon/logging/logging.go` - slog JSON handler with lumberjack rotation
- `internal/daemon/api/server.go` - Chi router with middleware stack
- `internal/daemon/api/middleware.go` - Request logging middleware
- `internal/daemon/api/handlers.go` - Health endpoint
- `go.mod` / `go.sum` - Added chi/v5, yaml.v3, lumberjack.v2 dependencies

### Decisions
- Logs write to both stderr and file (multiwriter) for visibility during development
- Health handler uses separate function instead of method on Server struct for simplicity
- Error from json.Encode is handled to satisfy linter (errcheck)
- Request logger logs duration in milliseconds rather than duration string
