package api

import (
	"log/slog"

	"github.com/kareemaly/cortex/internal/events"
	"github.com/kareemaly/cortex/internal/tmux"
)

// Dependencies holds all dependencies required by API handlers.
type Dependencies struct {
	StoreManager           *StoreManager
	ConclusionStoreManager *ConclusionStoreManager
	SessionManager         *SessionManager
	QueueManager           *QueueManager
	TmuxManager            *tmux.Manager
	Bus                    *events.Bus
	Logger                 *slog.Logger
	CortexdPath            string
	DefaultsDir            string
}
