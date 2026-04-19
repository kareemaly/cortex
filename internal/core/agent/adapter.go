// Package agent unifies per-agent status detection (Claude, Codex,
// OpenCode) behind a single Adapter struct + registry. An Adapter is plain
// data: function-pointer fields fill the same conceptual slots
// (ResolveTranscript, ParseTranscriptLine, AwaitingInputPhrases). This is
// deliberately NOT an interface — overwhelmingly what differs between
// agents is configuration, not behaviour, and struct fields are cheaper to
// override in tests than mock types.
//
// The detection model this package implements is intentionally simple:
//
//   - working: transcript line written OR pane-hash changed since last tick.
//   - awaiting_input: pane content contains one of the adapter's literal
//     phrases (plain substring match, no regex).
//   - idle: quiet on both channels for IdleWindow.
//
// The prior "box pattern" regex matcher (border + anchor) has been removed.
// Reference implementations (claude-squad, agentapi) get reliable results
// with raw-hash stability + one literal phrase; cortex does the same.
package agent

import (
	"bytes"
	"sort"
	"sync"
	"sync/atomic"
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
	IdleWindow       time.Duration
	DiscoveryTimeout time.Duration

	// AwaitingInputPhrases are literal substrings. If any appears in a pane
	// capture, the session is reported as awaiting_input. Keep the list short
	// and choose phrases that can't collide with tool stdout. Matching is
	// case-sensitive; include the exact casing from the agent's dialog.
	AwaitingInputPhrases []string

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
// program start. Also seeds the phrase-hit counter map so every declared
// phrase has a zero entry even before its first match (operators can then
// tell "phrase registered but never fired" from "phrase unknown").
func Register(a *Adapter) {
	if a == nil || a.Name == "" {
		panic("agent: Register requires a non-nil Adapter with a Name")
	}
	if _, exists := registry[a.Name]; exists {
		panic("agent: duplicate adapter registration: " + a.Name)
	}
	registry[a.Name] = a

	phraseHitsMu.Lock()
	defer phraseHitsMu.Unlock()
	if phraseHits[a.Name] == nil {
		phraseHits[a.Name] = make(map[string]*atomic.Int64)
	}
	for _, p := range a.AwaitingInputPhrases {
		if p == "" {
			continue
		}
		if _, ok := phraseHits[a.Name][p]; !ok {
			phraseHits[a.Name][p] = new(atomic.Int64)
		}
	}
}

// MatchAwaitingInput returns the first AwaitingInputPhrases entry present in
// content, or "" on no match. Plain substring, case-sensitive. Callers record
// a hit via RecordPhraseHit.
func (a *Adapter) MatchAwaitingInput(content []byte) string {
	for _, p := range a.AwaitingInputPhrases {
		if p == "" {
			continue
		}
		if bytes.Contains(content, []byte(p)) {
			return p
		}
	}
	return ""
}

// Get returns the adapter with the given name.
func Get(name string) (*Adapter, bool) {
	a, ok := registry[name]
	return a, ok
}

// All returns every registered adapter. Used by telemetry and tests that
// want to exercise every adapter's tables.
func All() []*Adapter {
	out := make([]*Adapter, 0, len(registry))
	for _, a := range registry {
		out = append(out, a)
	}
	return out
}

// ---------------- Phrase-hit telemetry ----------------
//
// Hit counters are keyed by (agent, phrase). Pointers are minted at Register
// time so the hot path can increment atomically without taking a mutex.

var (
	phraseHitsMu sync.Mutex
	phraseHits   = map[string]map[string]*atomic.Int64{}
)

// PhraseStats is a single (agent, phrase) → hits row. Returned by
// AllPhraseStats in a stable-sorted order for the debug endpoint.
type PhraseStats struct {
	Agent  string `json:"agent"`
	Phrase string `json:"phrase"`
	Hits   int64  `json:"hits"`
}

// RecordPhraseHit increments the counter for (agent, phrase). Phrases not
// registered under the named agent are ignored — this keeps the hot path
// lenient if the supervisor ever calls with a stale name.
func RecordPhraseHit(agent, phrase string) {
	phraseHitsMu.Lock()
	counters, ok := phraseHits[agent]
	phraseHitsMu.Unlock()
	if !ok {
		return
	}
	c, ok := counters[phrase]
	if !ok {
		return
	}
	c.Add(1)
}

// AllPhraseStats returns every registered (agent, phrase) pair and its hit
// count, sorted by agent then phrase so the debug payload is stable across
// calls.
func AllPhraseStats() []PhraseStats {
	phraseHitsMu.Lock()
	defer phraseHitsMu.Unlock()

	out := make([]PhraseStats, 0)
	for agent, phrases := range phraseHits {
		for phrase, c := range phrases {
			out = append(out, PhraseStats{
				Agent:  agent,
				Phrase: phrase,
				Hits:   c.Load(),
			})
		}
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Agent != out[j].Agent {
			return out[i].Agent < out[j].Agent
		}
		return out[i].Phrase < out[j].Phrase
	})
	return out
}
