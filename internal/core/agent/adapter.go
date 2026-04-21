// Package agent unifies per-agent status detection (Claude, Codex,
// OpenCode) behind a single Adapter struct + registry. An Adapter is plain
// data: function-pointer fields fill the same conceptual slots
// (ResolveTranscript, ParseTranscriptLine). This is deliberately NOT an
// interface — overwhelmingly what differs between agents is configuration,
// not behaviour, and struct fields are cheaper to override in tests than
// mock types.
//
// The detection model is authoritative transcript parsing + liveness:
//   - working: transcript line with activity flag set, OR no authoritative
//     status given (agent is implicitly working).
//   - awaiting_input: agent or plugin reports it explicitly in transcript.
//   - idle: not applicable; only transcript events and liveness matter.
//
// Pane-based status detection (phrase matching, idle-window decay) has been
// removed in favor of agentstatus Hook integration, which provides
// ground-truth status from the agent itself.
package agent

import (
	"time"

	"github.com/kareemaly/cortex/internal/session"
)

// TranscriptEvent is what ParseTranscriptLine returns for a single transcript
// line.
//
// Activity means "the agent produced output" — a liveness signal. Any
// non-empty line is activity for Claude; Codex and OpenCode only flag
// activity when they have a specific event to report. Activity alone is
// enough for the decision machine to hold / promote to working.
//
// Status, when non-empty, is authoritative: the transcript event speaks
// directly to status (OpenCode plugin pushing "awaiting_input"; Codex
// task_started/task_complete). Authoritative status overrides pane signals
// in the decision machine — the agent knows its own state better than we can
// infer from a screen scrape.
//
// Tool and Work are forwarded verbatim for display in the dashboard.
type TranscriptEvent struct {
	Activity bool
	Status   session.AgentStatus
	Tool     string
	Work     string
	IsError  bool
}

// RuntimeCtx is passed to ResolveTranscript during the discovery phase.
// Adapters use TranscriptHint plus process-env to locate the transcript
// file.
type RuntimeCtx struct {
	SessionID      string
	WorkingDir     string
	TranscriptHint string
	Env            map[string]string
}

// Adapter is the per-agent unit of configuration. Each supported agent owns
// one file under internal/core/agent (adapter_claude.go etc.) that populates
// and registers an Adapter in its package init.
type Adapter struct {
	Name             string
	DiscoveryTimeout time.Duration

	ResolveTranscript   func(RuntimeCtx) string
	ParseTranscriptLine func(line []byte) TranscriptEvent
}

// ---------------- Registry ----------------
//
// Adapter files call Register from their init() blocks. The registry is
// written to only at package load time and read concurrently thereafter,
// which Go's memory model permits for an unsynchronized read-only map.

var registry = map[string]*Adapter{}

// Register installs an adapter under its Name. Calling Register with a
// duplicate name panics — a misconfigured init() should fail loudly at
// program start.
func Register(a *Adapter) {
	if a == nil || a.Name == "" {
		panic("agent: Register requires a non-nil Adapter with a Name")
	}
	if _, exists := registry[a.Name]; exists {
		panic("agent: duplicate adapter registration: " + a.Name)
	}
	registry[a.Name] = a
}


// Get returns the adapter with the given name.
func Get(name string) (*Adapter, bool) {
	a, ok := registry[name]
	return a, ok
}

// All returns every registered adapter. Used by tests that
// want to exercise every adapter's tables.
func All() []*Adapter {
	out := make([]*Adapter, 0, len(registry))
	for _, a := range registry {
		out = append(out, a)
	}
	return out
}
