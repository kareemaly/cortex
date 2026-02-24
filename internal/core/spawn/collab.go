package spawn

import (
	"context"
)

// CollabSpawnRequest contains parameters for spawning a collab session.
type CollabSpawnRequest struct {
	CollabID      string
	Repo          string
	Prompt        string
	ArchitectPath string
	TmuxSession   string
	Agent         string
	Companion     string
	AgentArgs     []string
	TicketsDir    string
}

// CollabSpawnResult contains the result of a collab spawn operation.
type CollabSpawnResult struct {
	CollabID    string
	TmuxWindow  string
	TmuxSession string
}

// SpawnCollab spawns a new collab agent session.
// Unlike ticket spawning, there is no orphan/resume concept — always spawns fresh.
func (s *Spawner) SpawnCollab(ctx context.Context, req CollabSpawnRequest) (*CollabSpawnResult, error) {
	result, err := s.Spawn(ctx, SpawnRequest{
		AgentType:     AgentTypeCollabAgent,
		Agent:         req.Agent,
		TmuxSession:   req.TmuxSession,
		ArchitectPath: req.ArchitectPath,
		TicketsDir:    req.TicketsDir,
		CollabID:      req.CollabID,
		Prompt:        req.Prompt,
		Repo:          req.Repo,
		Companion:     req.Companion,
		AgentArgs:     req.AgentArgs,
	})
	if err != nil {
		return nil, err
	}

	if !result.Success {
		return nil, &ConfigError{Field: "spawn", Message: result.Message}
	}

	return &CollabSpawnResult{
		CollabID:    req.CollabID,
		TmuxWindow:  result.TmuxWindow,
		TmuxSession: req.TmuxSession,
	}, nil
}
