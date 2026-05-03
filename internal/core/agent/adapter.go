// Package agent provides per-session status supervision. The agentruntime
// ingest receiver (hook-based) is the sole source of agent status;
// transcript parsing has been removed. Liveness detection is the only
// independent concern: it monitors process death and ends the cortex
// session record when the agent disappears.
package agent

import "github.com/kareemaly/cortex/internal/session"

// HubEvent carries the status fields from a normalized agentruntime Event
// that the supervisor needs to forward to /agent/status.
type HubEvent struct {
	Status session.AgentStatus
	Tool   string
	Work   string
}
