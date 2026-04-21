package agent

import (
	"time"

	"github.com/kareemaly/cortex/internal/session"
)

// Source tags which part of the supervisor produced a Signal. The decision
// machine reads Source to route — e.g. liveness always wins, transcript with
// an explicit status takes precedence.
type Source string

const (
	SourceTranscript Source = "transcript"
	SourceLiveness   Source = "liveness"
)

// Signal is the internal event fed to Decision.Apply. One signal per tick
// per input source — the supervisor fans transcript / liveness into a
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
}

// DecisionConfig seeds the starting state.
type DecisionConfig struct {
	InitialStatus session.AgentStatus
}

// Decision is the single writer of a session's current status. One lives per
// supervisor; Apply is called sequentially from the decision goroutine so
// there is no concurrent mutation.
type Decision struct {
	cfg DecisionConfig

	current session.AgentStatus
	tool    *string
	work    *string
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
//  4. Source==transcript && Activity → working
//
// Only authoritative transcript signals and liveness matter — pane-based
// idle decay was removed after integration with agentstatus Hub.
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
		case s.Status != "":
			// Authoritative transcript status — issued by plugin (opencode) or
			// agent event (codex). With one carve-out: a plugin that belatedly
			// says "idle" while already in awaiting_input shouldn't undo it —
			// in practice they agree within a tick, but edge case protection.
			if d.current == session.AgentStatusAwaitingInput && s.Status == session.AgentStatusIdle {
				// Keep awaiting_input; transcript agreement will follow.
			} else {
				d.current = s.Status
			}
		case s.Activity:
			// Activity without explicit status is a liveness beat. Promote
			// out of starting/idle, hold awaiting_input (user is still
			// blocking — the transcript line is just the agent acknowledging
			// something internally).
			if d.current != session.AgentStatusAwaitingInput {
				d.current = session.AgentStatusWorking
			}
		}
	}

	return d.current, d.current != prev
}
