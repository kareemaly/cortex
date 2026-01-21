package api

import (
	"log/slog"

	"github.com/kareemaly/cortex1/internal/lifecycle"
	"github.com/kareemaly/cortex1/internal/tmux"
)

// Dependencies holds all dependencies required by API handlers.
type Dependencies struct {
	StoreManager *StoreManager
	TmuxManager  *tmux.Manager
	HookExecutor *lifecycle.Executor
	Logger       *slog.Logger
}
