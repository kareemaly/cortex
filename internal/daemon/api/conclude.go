package api

import (
	"log/slog"
	"time"

	architectconfig "github.com/kareemaly/cortex/internal/architect/config"
	"github.com/kareemaly/cortex/internal/tmux"
)

// ConcludeParams holds parameters for creating a conclusion and killing a tmux window.
type ConcludeParams struct {
	ProjectPath   string
	EntityType    string
	EntityID      string
	TmuxWindow    string
	Content       string
	StartedAt     time.Time
	Repo          string
	Prompt        string
	Logger        *slog.Logger
	TmuxManager   *tmux.Manager
	ConclusionMgr *ConclusionStoreManager
}

// CreateConclusionAndKillWindow creates a conclusion record and kills the associated tmux window.
// Returns the conclusion ID if created, or empty string.
func CreateConclusionAndKillWindow(params ConcludeParams) string {
	var conclusionID string

	// Create conclusion record
	if params.ConclusionMgr != nil {
		conclusionStore, err := params.ConclusionMgr.GetStore(params.ProjectPath)
		if err == nil {
			rec, createErr := conclusionStore.Create(
				params.EntityType,
				params.EntityID,
				params.Repo,
				params.Content,
				params.StartedAt,
				params.Prompt,
			)
			if createErr != nil {
				params.Logger.Warn("failed to create conclusion", "type", params.EntityType, "error", createErr)
			} else {
				conclusionID = rec.ID
			}
		}
	}

	// Kill tmux window if associated (best-effort)
	if params.TmuxWindow != "" && params.TmuxManager != nil {
		projectCfg, _ := architectconfig.Load(params.ProjectPath)
		tmuxSession := projectCfg.GetTmuxSessionName()

		if killErr := params.TmuxManager.KillWindow(tmuxSession, params.TmuxWindow); killErr != nil {
			if !tmux.IsWindowNotFound(killErr) && !tmux.IsSessionNotFound(killErr) {
				params.Logger.Warn("failed to kill tmux window", "type", params.EntityType, "window", params.TmuxWindow, "error", killErr)
			}
		}
	}

	return conclusionID
}
