package api

import (
	"context"
	"log/slog"

	"github.com/kareemaly/cortex/internal/events"
	"github.com/kareemaly/cortex/internal/tmux"
	"github.com/kareemaly/cortex/internal/tmux/observer"
)

// Dependencies holds all dependencies required by API handlers.
//
// SupervisorCtx is the daemon's root context. Long-lived goroutines started
// from HTTP handlers (notably agent-status supervisors spawned via
// POST /tickets/.../spawn) must bind to this context rather than the HTTP
// request context — otherwise they die the moment the handler returns.
type Dependencies struct {
	StoreManager           *StoreManager
	ConclusionStoreManager *ConclusionStoreManager
	SessionManager         *SessionManager
	TmuxManager            *tmux.Manager
	PaneObserver           *observer.Observer
	Bus                    *events.Bus
	Logger                 *slog.Logger
	SupervisorCtx          context.Context
	CortexdPath            string
	DefaultsDir            string
	HubManager             *HubManager
}
