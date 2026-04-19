// Package agent unifies per-agent status detection (Claude, Codex,
// OpenCode) behind a single Adapter struct + registry. An Adapter is
// plain data: function-pointer fields fill the same conceptual slots
// (ResolveTranscript, ParseLine, PanePatterns). This is deliberately NOT
// an interface — overwhelmingly what differs between agents is
// configuration, not behaviour, and struct fields are cheaper to override
// in tests than mock types.
package agent

import (
	"time"

	"github.com/kareemaly/cortex/internal/session"
)

// StatusUpdate is what ParseLine returns for a single transcript line.
// An empty Status means "ignore this line". Tool and Work are forwarded
// to the daemon verbatim.
type StatusUpdate struct {
	Status session.AgentStatus
	Tool   string
	Work   string
}

// RuntimeCtx is passed to ResolveTranscript during the discovery phase.
// Adapters use TranscriptHint plus process-env to locate the transcript
// file. WorkingDir is repeated here for convenience.
type RuntimeCtx struct {
	SessionID      string
	WorkingDir     string
	TranscriptHint string
	Env            map[string]string
}

// Adapter is the per-agent unit of configuration. Each supported agent
// owns one file under internal/core/agent (adapter_claude.go etc.) that
// populates and registers an Adapter in its package init.
type Adapter struct {
	Name             string
	IdleThreshold    time.Duration
	DiscoveryTimeout time.Duration

	ResolveTranscript func(RuntimeCtx) string
	ParseLine         func(line []byte) StatusUpdate

	PanePatterns PanePatterns
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

// All returns every registered adapter. Primarily used by telemetry and
// tests that want to exercise every adapter's tables.
func All() []*Adapter {
	out := make([]*Adapter, 0, len(registry))
	for _, a := range registry {
		out = append(out, a)
	}
	return out
}
