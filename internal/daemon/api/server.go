package api

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

// Server represents the HTTP server for the daemon.
type Server struct {
	httpServer *http.Server
	logger     *slog.Logger
}

// NewRouter creates a chi router with all API routes configured.
func NewRouter(deps *Dependencies, logger *slog.Logger) chi.Router {
	r := chi.NewRouter()

	// Global middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(RequestLogger(logger))

	// Global endpoints (no project required)
	r.Get("/health", HealthHandler())
	r.Get("/projects", ProjectsHandler(deps.StoreManager))
	r.Post("/daemon/focus", DaemonFocusHandler(deps.TmuxManager))

	// Project-scoped routes
	r.Group(func(r chi.Router) {
		r.Use(ProjectRequired())

		// SSE event stream
		eventHandlers := NewEventHandlers(deps)
		r.Get("/events", eventHandlers.Stream)

		// Ticket routes
		ticketHandlers := NewTicketHandlers(deps)
		r.Route("/tickets", func(r chi.Router) {
			r.Get("/", ticketHandlers.ListAll)
			r.Post("/", ticketHandlers.Create)
			r.Get("/by-id/{id}", ticketHandlers.GetByID)
			r.Get("/{status}", ticketHandlers.ListByStatus)
			r.Get("/{status}/{id}", ticketHandlers.Get)
			r.Put("/{status}/{id}", ticketHandlers.Update)
			r.Delete("/{status}/{id}", ticketHandlers.Delete)
			r.Post("/{status}/{id}/move", ticketHandlers.Move)
			r.Post("/{status}/{id}/spawn", ticketHandlers.Spawn)
			r.Post("/{id}/comments", ticketHandlers.AddComment)
			r.Post("/{id}/reviews", ticketHandlers.RequestReview)
			r.Post("/{id}/focus", ticketHandlers.Focus)
			r.Post("/{id}/conclude", ticketHandlers.Conclude)
			r.Post("/{id}/comments/{comment_id}/execute", ticketHandlers.ExecuteAction)
		})

		// Architect routes
		architectHandlers := NewArchitectHandlers(deps)
		r.Route("/architect", func(r chi.Router) {
			r.Get("/", architectHandlers.GetState)
			r.Post("/spawn", architectHandlers.Spawn)
			r.Post("/focus", architectHandlers.Focus)
		})

		// Session routes
		sessionHandlers := NewSessionHandlers(deps)
		r.Route("/sessions", func(r chi.Router) {
			r.Delete("/{id}", sessionHandlers.Kill)
			r.Post("/{id}/approve", sessionHandlers.Approve)
		})

		// Agent routes
		agentHandlers := NewAgentHandlers(deps)
		r.Route("/agent", func(r chi.Router) {
			r.Post("/status", agentHandlers.UpdateStatus)
		})

		// Prompt routes
		promptHandlers := NewPromptHandlers(deps)
		r.Get("/prompts/resolve", promptHandlers.Resolve)
	})

	return r
}

// NewServer creates a new Server with the given configuration.
func NewServer(port int, logger *slog.Logger, deps *Dependencies) *Server {
	r := NewRouter(deps, logger)

	httpServer := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      r,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 0, // No timeout for SSE support
		IdleTimeout:  60 * time.Second,
	}

	return &Server{
		httpServer: httpServer,
		logger:     logger,
	}
}

// Run starts the server and blocks until the context is cancelled.
// It performs graceful shutdown when the context is done.
func (s *Server) Run(ctx context.Context) error {
	errCh := make(chan error, 1)

	go func() {
		s.logger.Info("starting server", "addr", s.httpServer.Addr)
		if err := s.httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
		s.logger.Info("shutting down...")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()

		if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
			return fmt.Errorf("server shutdown failed: %w", err)
		}

		s.logger.Info("server stopped")
		return nil
	}
}
