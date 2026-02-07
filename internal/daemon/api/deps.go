package api

import (
	"log/slog"

	"github.com/kareemaly/cortex/internal/events"
	"github.com/kareemaly/cortex/internal/tmux"
)

// Dependencies holds all dependencies required by API handlers.
type Dependencies struct {
	StoreManager     *StoreManager
	DocsStoreManager *DocsStoreManager
	SessionManager   *SessionManager
	TmuxManager      *tmux.Manager
	Bus              *events.Bus
	Logger           *slog.Logger
	CortexdPath      string // Optional: path to cortexd binary for spawn operations
}
