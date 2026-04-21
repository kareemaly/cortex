package spawn

import (
	"context"
	"log/slog"

	"github.com/kareemaly/cortex/internal/core/agent"
	daemonconfig "github.com/kareemaly/cortex/internal/daemon/config"
)

// agentSupervisorParams is the unified input for starting a per-session
// status supervisor. Agent-specific discovery inputs (claude's transcript
// path, codex's $CODEX_HOME, opencode's status file) are all expressed as
// TranscriptHint — each adapter's ResolveTranscript knows how to turn the
// hint into the real transcript path.
type agentSupervisorParams struct {
	Agent          string // "claude" | "codex" | "opencode"
	TranscriptHint string
	LivenessPath   string
	SessionID      string
	TicketID       string
	ArchitectPath  string
	Logger         *slog.Logger
}

// startAgentSupervisor wires the adapter-based supervisor for one agent
// session. It returns (nil, nil) when supervision isn't possible (missing
// required inputs or no adapter registered); the caller then runs the
// session unsupervised rather than failing the spawn. Any diagnostic is
// logged so the skip isn't silent.
func startAgentSupervisor(ctx context.Context, p agentSupervisorParams) (context.CancelFunc, error) {
	if p.Agent == "" || p.TranscriptHint == "" || p.LivenessPath == "" {
		return nil, nil
	}
	if p.SessionID == "" && p.TicketID == "" {
		if p.Logger != nil {
			p.Logger.Warn("agent supervisor skipped: both SessionID and TicketID empty",
				"agent", p.Agent, "architect_path", p.ArchitectPath)
		}
		return nil, nil
	}
	adapter, ok := agent.Get(p.Agent)
	if !ok {
		if p.Logger != nil {
			p.Logger.Warn("agent supervisor skipped: no adapter registered", "agent", p.Agent)
		}
		return nil, nil
	}
	if p.Logger == nil {
		p.Logger = slog.Default()
	}
	return agent.StartSupervisor(ctx, agent.SupervisorConfig{
		SessionID:     p.SessionID,
		TicketID:      p.TicketID,
		ArchitectPath: p.ArchitectPath,
		LivenessPath:  p.LivenessPath,
		Adapter:       adapter,
		Runtime:       agent.RuntimeCtx{TranscriptHint: p.TranscriptHint},
		DaemonURL:     daemonconfig.DefaultDaemonURL,
		Logger:        p.Logger,
	})
}
