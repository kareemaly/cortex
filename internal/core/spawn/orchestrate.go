package spawn

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"

	projectconfig "github.com/kareemaly/cortex/internal/project/config"
	"github.com/kareemaly/cortex/internal/session"
	"github.com/kareemaly/cortex/internal/ticket"
)

// OrchestrateStore defines the ticket store operations for orchestration.
type OrchestrateStore interface {
	Get(id string) (*ticket.Ticket, ticket.Status, error)
	Move(id string, to ticket.Status) error
}

// OrchestrateRequest contains parameters for orchestrating a spawn operation.
type OrchestrateRequest struct {
	TicketID    string
	Mode        string // "normal", "resume", "fresh" (validated internally; defaults to "normal")
	Agent       string // optional: falls back to project config, then "claude"
	ProjectPath string
	TicketsDir  string // optional: derived from ProjectPath if empty
	TmuxSession string // optional: derived from project config name if empty
}

// Outcome describes the result of an orchestration.
type Outcome string

const (
	// OutcomeSpawned indicates a new session was spawned.
	OutcomeSpawned Outcome = "spawned"
	// OutcomeResumed indicates an orphaned session was resumed.
	OutcomeResumed Outcome = "resumed"
	// OutcomeAlreadyActive indicates the ticket already has an active session.
	OutcomeAlreadyActive Outcome = "already_active"
)

// OrchestrateResult contains the result of a successful orchestration.
type OrchestrateResult struct {
	Outcome      Outcome
	Ticket       *ticket.Ticket
	TicketStatus ticket.Status
	SpawnResult  *SpawnResult // nil when Outcome is AlreadyActive
	StateInfo    *StateInfo
	TmuxSession  string // resolved tmux session name
}

// OrchestrateDeps contains the external dependencies for orchestration.
type OrchestrateDeps struct {
	Store        OrchestrateStore
	SessionStore SessionStoreInterface
	TmuxManager  TmuxManagerInterface
	CortexdPath  string       // optional: empty means auto-discover via binpath
	Logger       *slog.Logger // optional
}

// Orchestrate is the single source of truth for spawning ticket agent sessions.
// Both the HTTP API handler and MCP tool call this function.
//
// State/mode matrix:
//
//	| Mode    | Normal      | Active         | Orphaned    |
//	|---------|-------------|----------------|-------------|
//	| normal  | Spawn new   | AlreadyActive  | StateError  |
//	| resume  | StateError  | StateError     | Resume      |
//	| fresh   | StateError  | StateError     | Fresh       |
func Orchestrate(ctx context.Context, req OrchestrateRequest, deps OrchestrateDeps) (*OrchestrateResult, error) {
	// 1. Validate mode
	if req.Mode == "" {
		req.Mode = "normal"
	}
	if req.Mode != "normal" && req.Mode != "resume" && req.Mode != "fresh" {
		return nil, &ConfigError{Field: "Mode", Message: "must be 'normal', 'resume', or 'fresh'"}
	}

	// 2. Validate TmuxManager
	if deps.TmuxManager == nil {
		return nil, &ConfigError{Field: "TmuxManager", Message: "tmux manager is required"}
	}

	// 3. Load project config
	projectCfg, err := projectconfig.Load(req.ProjectPath)
	if err != nil {
		return nil, &ConfigError{Field: "ProjectPath", Message: "failed to load project config: " + err.Error()}
	}

	// 4. Resolve ticket type and agent config
	t, ticketStatus, err := deps.Store.Get(req.TicketID)
	if err != nil {
		return nil, err
	}

	ticketType := t.Type
	if ticketType == "" {
		ticketType = ticket.DefaultTicketType
	}

	ticketRoleCfg, ticketCfgErr := projectCfg.TicketRoleConfig(ticketType)
	if ticketCfgErr != nil {
		return nil, &ConfigError{Field: "ticket." + ticketType, Message: ticketCfgErr.Error()}
	}

	agent := req.Agent
	if agent == "" {
		agent = string(ticketRoleCfg.Agent)
	}
	if agent == "" {
		agent = "claude"
	}

	// 5. Resolve TmuxSession: request > project config name
	tmuxSession := req.TmuxSession
	if tmuxSession == "" {
		tmuxSession = projectCfg.Name
	}
	if tmuxSession == "" {
		return nil, &ConfigError{Field: "TmuxSession", Message: "tmux session name is required (set in project config or pass explicitly)"}
	}

	// 6. Resolve TicketsDir
	ticketsDir := req.TicketsDir
	if ticketsDir == "" {
		ticketsDir = filepath.Join(req.ProjectPath, ".cortex", "tickets")
	}

	// 7. Look up existing session
	var existingSess *session.Session
	if deps.SessionStore != nil {
		existingSess, _ = deps.SessionStore.GetByTicketID(t.ID)
	}

	// 8. Detect state
	stateInfo, err := DetectTicketState(existingSess, tmuxSession, deps.TmuxManager)
	if err != nil {
		return nil, err
	}

	// 9. State/mode matrix
	spawner := NewSpawner(Dependencies{
		Store:        deps.Store,
		SessionStore: deps.SessionStore,
		TmuxManager:  deps.TmuxManager,
		CortexdPath:  deps.CortexdPath,
		Logger:       deps.Logger,
	})

	useWorktree := projectCfg.Git.Worktrees

	buildSpawnReq := func() SpawnRequest {
		return SpawnRequest{
			AgentType:      AgentTypeTicketAgent,
			Agent:          agent,
			TmuxSession:    tmuxSession,
			ProjectPath:    req.ProjectPath,
			TicketsDir:     ticketsDir,
			TicketID:       req.TicketID,
			Ticket:         t,
			UseWorktree:    useWorktree,
			AgentArgs:      ticketRoleCfg.Args,
			BaseConfigPath: projectCfg.ResolvedExtendPath(),
		}
	}

	var result *SpawnResult
	var outcome Outcome

	switch stateInfo.State {
	case StateNormal:
		switch req.Mode {
		case "normal":
			result, err = spawner.Spawn(ctx, buildSpawnReq())
			outcome = OutcomeSpawned
		case "resume":
			return nil, &StateError{TicketID: req.TicketID, State: StateNormal, Message: "cannot resume - no existing session to resume"}
		case "fresh":
			return nil, &StateError{TicketID: req.TicketID, State: StateNormal, Message: "cannot use fresh mode - no existing session to clear"}
		}

	case StateActive:
		if req.Mode == "normal" {
			return &OrchestrateResult{
				Outcome:      OutcomeAlreadyActive,
				Ticket:       t,
				TicketStatus: ticketStatus,
				StateInfo:    stateInfo,
				TmuxSession:  tmuxSession,
			}, nil
		}
		return nil, &StateError{
			TicketID: req.TicketID,
			State:    StateActive,
			Message:  "session is currently active - wait for it to finish or close the tmux window",
		}

	case StateOrphaned:
		switch req.Mode {
		case "normal":
			return nil, &StateError{
				TicketID: req.TicketID,
				State:    StateOrphaned,
				Message:  "session was orphaned (tmux window closed). Use mode='resume' to continue or mode='fresh' to start over",
			}
		case "resume":
			if stateInfo.Session == nil {
				return nil, &StateError{
					TicketID: req.TicketID,
					State:    StateOrphaned,
					Message:  "cannot resume - no session stored",
				}
			}
			result, err = spawner.Resume(ctx, ResumeRequest{
				AgentType:   AgentTypeTicketAgent,
				Agent:       agent,
				TmuxSession: tmuxSession,
				ProjectPath: req.ProjectPath,
				TicketsDir:  ticketsDir,
				SessionID:   "resume", // sessions are ephemeral, use a placeholder for resume
				WindowName:  stateInfo.Session.TmuxWindow,
				TicketID:    req.TicketID,
				AgentArgs:   ticketRoleCfg.Args,
			})
			outcome = OutcomeResumed
		case "fresh":
			result, err = spawner.Fresh(ctx, buildSpawnReq())
			outcome = OutcomeSpawned
		}
	}

	if err != nil {
		return nil, err
	}

	// Handle soft failures from spawner (e.g., prompt load failure, tmux spawn failure)
	if result == nil || !result.Success {
		msg := "spawn operation failed"
		if result != nil && result.Message != "" {
			msg = result.Message
		}
		return nil, fmt.Errorf("spawn: %s", msg)
	}

	// 10. Post-spawn: move ticket to progress if in backlog
	if ticketStatus == ticket.StatusBacklog {
		if moveErr := deps.Store.Move(req.TicketID, ticket.StatusProgress); moveErr != nil {
			if deps.Logger != nil {
				deps.Logger.Warn("failed to move ticket to progress", "error", moveErr)
			}
		}
	}

	// 11. Re-read ticket to get updated state
	t, ticketStatus, err = deps.Store.Get(req.TicketID)
	if err != nil {
		return nil, err
	}

	return &OrchestrateResult{
		Outcome:      outcome,
		Ticket:       t,
		TicketStatus: ticketStatus,
		SpawnResult:  result,
		StateInfo:    stateInfo,
		TmuxSession:  tmuxSession,
	}, nil
}
