package api

import (
	"context"
	"log/slog"

	"github.com/kareemaly/cortex/internal/events"
	"github.com/kareemaly/cortex/internal/tmux"
)

type Dependencies struct {
	StoreManager    *StoreManager
	SessionManager  *SessionManager
	TmuxManager     *tmux.Manager
	Bus             *events.Bus
	Logger          *slog.Logger
	SupervisorCtx   context.Context
	CortexdPath     string
	DefaultsDir     string
	ReceiverManager *ReceiverManager
	DaemonEndpoint  string
}
