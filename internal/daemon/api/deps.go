package api

import (
	"log/slog"

	"github.com/kareemaly/cortex/internal/tmux"
)

// Dependencies holds all dependencies required by API handlers.
type Dependencies struct {
	StoreManager *StoreManager
	TmuxManager  *tmux.Manager
	Logger       *slog.Logger
}
