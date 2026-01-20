package api

import (
	"log/slog"

	"github.com/kareemaly/cortex1/internal/lifecycle"
	projectconfig "github.com/kareemaly/cortex1/internal/project/config"
	"github.com/kareemaly/cortex1/internal/ticket"
	"github.com/kareemaly/cortex1/internal/tmux"
)

// Dependencies holds all dependencies required by API handlers.
type Dependencies struct {
	TicketStore   *ticket.Store
	ProjectConfig *projectconfig.Config
	ProjectRoot   string
	TmuxManager   *tmux.Manager
	HookExecutor  *lifecycle.Executor
	Logger        *slog.Logger
}
