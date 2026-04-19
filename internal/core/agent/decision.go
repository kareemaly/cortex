package agent

import (
	"time"

	"github.com/kareemaly/cortex/internal/session"
)

// Source tags which part of the supervisor produced a Signal. The decision
// machine reads Source to route — e.g. liveness always wins, transcript with
// an explicit status overrides pane signals.
type Source string

const (
	SourceTranscript Source = "transcript"
	SourcePane       Source = "pane"
	SourceLiveness   Source = "liveness"
)

// Signal is the internal event fed to Decision.Apply. One signal per tick
// per input source — the supervisor fans transcript / pane / liveness into a
// single stream and the decision machine is the sole writer of status.
type Signal struct {
	Source Source
	At     time.Time

	// Activity, Status, Tool, Work, IsError: populated when Source==transcript.
	Activity bool
	Status   session.AgentStatus
	Tool     *string
	Work     *string
	IsError  bool

	// HasAwaitingPhrase, Changed: populated when Source==pane.
	// HasAwaitingPhrase is true when the pane content contains one of the
	// adapter's AwaitingInputPhrases. Changed is true when the pane hash
	// moved since the prior tick.
	HasAwaitingPhrase bool
	Changed           bool
}

// DecisionConfig seeds the starting state and picks the idle window — how
// long without transcript activity OR pane change before flipping to idle.
// Zero IdleWindow disables time-based idle (the status then lives entirely
// on authoritative transcript signals).
type DecisionConfig struct {
	InitialStatus session.AgentStatus
	IdleWindow    time.Duration
}

// Decision is the single writer of a session's current status. One lives per
// supervisor; Apply is called sequentially from the decision goroutine so
// there is no concurrent mutation.
type Decision struct {
	cfg DecisionConfig

	current session.AgentStatus
	tool    *string
	work    *string

	lastActivityAt time.Time // last transcript activity OR pane change
}

// NewDecision seeds state from the config's InitialStatus (defaults to
// starting).
func NewDecision(cfg DecisionConfig) *Decision {
	initial := cfg.InitialStatus
	if initial == "" {
		initial = session.AgentStatusStarting
	}
	return &Decision{cfg: cfg, current: initial}
}

// Current returns the decision machine's current status snapshot. Safe only
// from the decision goroutine.
func (d *Decision) Current() session.AgentStatus { return d.current }

// Tool returns the last known tool reported by the adapter.
func (d *Decision) Tool() *string { return d.tool }

// Work returns the last known work descriptor reported by the adapter.
func (d *Decision) Work() *string { return d.work }

// Apply feeds one Signal into the state machine and returns the resulting
// status along with whether that's a transition from the previous status.
//
// Precedence (highest wins, evaluated per signal):
//  1. Source==liveness → ended (terminal)
//  2. Source==transcript && IsError → error
//  3. Source==transcript && Status != "" → that status (authoritative:
//     opencode plugin, codex task events)
//  4. Source==pane && HasAwaitingPhrase → awaiting_input
//  5. Source==pane && Changed OR Source==transcript && Activity → working
//     (unless HasAwaitingPhrase overrode us above)
//  6. Source==pane && quiet for ≥ IdleWindow → idle
//
// The previous design carried a mess of "stable pane seen" flags plus a
// conjunction timer rule; that rule fired idle on a stable-pane-without-
// dialog-box heuristic which turned silent tool calls into false idles.
// This model drops the heuristic entirely — idle is computed from elapsed
// time since any signal of life, nothing more.
func (d *Decision) Apply(s Signal) (session.AgentStatus, bool) {
	prev := d.current

	// Once ended, stay ended — late transcript writes from a killed agent (or
	// a race between liveness-disappear and in-flight stdout) must not
	// resurrect the session.
	if d.current == session.AgentStatusEnded {
		return d.current, false
	}

	switch s.Source {
	case SourceLiveness:
		d.current = session.AgentStatusEnded

	case SourceTranscript:
		if s.Tool != nil {
			d.tool = s.Tool
		}
		if s.Work != nil {
			d.work = s.Work
		}
		switch {
		case s.IsError:
			d.current = session.AgentStatusError
			d.lastActivityAt = s.At
		case s.Status != "":
			// Authoritative transcript status wins over pane-derived state,
			// with one carve-out: a plugin that belatedly says "idle" while a
			// pane-matched permission dialog is still visible shouldn't undo
			// the awaiting_input flip. In practice plugins and panes agree
			// within a tick or two, so this only bites at the edges.
			if d.current == session.AgentStatusAwaitingInput && s.Status == session.AgentStatusIdle {
				// Keep awaiting_input; wait for the pane to clear first.
			} else {
				d.current = s.Status
			}
			d.lastActivityAt = s.At
		case s.Activity:
			// Activity without explicit status is a liveness beat. Promote
			// out of starting/idle, hold awaiting_input (user is still
			// blocking — the transcript line is just the agent acknowledging
			// something internally).
			if d.current != session.AgentStatusAwaitingInput {
				d.current = session.AgentStatusWorking
			}
			d.lastActivityAt = s.At
		}

	case SourcePane:
		switch {
		case s.HasAwaitingPhrase:
			d.current = session.AgentStatusAwaitingInput
			// Don't update lastActivityAt: the phrase is visible precisely
			// because the agent has stopped producing output.
		case s.Changed:
			// The pane moved. If we were awaiting_input and the dialog just
			// cleared, pane change counts as the agent resuming work.
			d.current = session.AgentStatusWorking
			d.lastActivityAt = s.At
		default:
			// No phrase, no change. Evaluate idle-by-timeout. Only `working`
			// can decay into `idle` — starting stays until something
			// happens; awaiting_input holds until the pane clears;
			// error/ended are terminal enough to not bother.
			if d.current != session.AgentStatusWorking {
				break
			}
			if d.cfg.IdleWindow == 0 {
				break
			}
			if d.lastActivityAt.IsZero() {
				break
			}
			if s.At.Sub(d.lastActivityAt) >= d.cfg.IdleWindow {
				d.current = session.AgentStatusIdle
			}
		}
	}

	return d.current, d.current != prev
}
