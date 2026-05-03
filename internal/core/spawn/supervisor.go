package spawn

import (
	"context"
	"log/slog"

	"github.com/kareemaly/cortex/internal/core/agent"
	daemonconfig "github.com/kareemaly/cortex/internal/daemon/config"
)

// agentSupervisorParams is the unified input for starting a per-session
// status supervisor. The supervisor monitors liveness (process death) and
// forwards Hub events to SSE.
type agentSupervisorParams struct {
	LivenessPath  string
	SessionID     string
	TicketID      string
	ArchitectPath string

	// HubEventSource, when non-nil, creates a filtered Hub event channel for
	// the given cortex session ID. Nil means liveness-only supervision.
	HubEventSource func(ctx context.Context, sessionID string) <-chan agent.HubEvent

	Logger *slog.Logger
}

// startAgentSupervisor wires the supervisor for one agent session. It returns
// (nil, nil) when supervision is not possible (missing LivenessPath or both
// IDs empty); the caller then runs the session unsupervised rather than
// failing the spawn. Diagnostics are logged so the skip isn't silent.
func startAgentSupervisor(ctx context.Context, p agentSupervisorParams) (context.CancelFunc, error) {
	if p.LivenessPath == "" {
		return nil, nil
	}
	if p.SessionID == "" && p.TicketID == "" {
		if p.Logger != nil {
			p.Logger.Warn("agent supervisor skipped: both SessionID and TicketID empty",
				"architect_path", p.ArchitectPath)
		}
		return nil, nil
	}
	if p.Logger == nil {
		p.Logger = slog.Default()
	}

	var hubEventSource func(ctx context.Context) <-chan agent.HubEvent
	if p.HubEventSource != nil && p.SessionID != "" {
		sessionID := p.SessionID
		hubEventSource = func(ctx context.Context) <-chan agent.HubEvent {
			return p.HubEventSource(ctx, sessionID)
		}
	}

	return agent.StartSupervisor(ctx, agent.SupervisorConfig{
		SessionID:      p.SessionID,
		TicketID:       p.TicketID,
		ArchitectPath:  p.ArchitectPath,
		LivenessPath:   p.LivenessPath,
		HubEventSource: hubEventSource,
		DaemonURL:      daemonconfig.DefaultDaemonURL,
		Logger:         p.Logger,
	})
}
