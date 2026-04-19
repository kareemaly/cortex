package agent

import (
	"time"

	"github.com/kareemaly/cortex/internal/session"
)

// Source tags which part of the supervisor produced a Signal. The
// decision machine routes on Source — e.g. only pane signals can flip
// the session to awaiting_input, regardless of what the transcript says.
type Source string

const (
	SourceTranscript Source = "transcript"
	SourcePane       Source = "pane"
	SourceLiveness   Source = "liveness"
	SourceTimer      Source = "timer"
)

// Signal is the internal event fed to Decision.Apply. It's the union of
// everything the supervisor's inputs (transcript tailer, pane observer,
// liveness watcher, timer) can produce.
type Signal struct {
	Source  Source
	Status  session.AgentStatus
	Tool    *string
	Work    *string
	Stable  bool // Source==pane: pane hash unchanged for the stability window
	HasBox  bool // Source==pane: a box pattern matched on this stable tail
	IsError bool // Source==transcript: explicit transcript error event
	At      time.Time
}

// DecisionConfig picks an initial status and the idle threshold — the
// minimum silence window before transcript silence alone can flip to
// idle. Zero IdleThreshold disables time-based idle (codex, which emits
// task_complete explicitly).
type DecisionConfig struct {
	InitialStatus session.AgentStatus
	IdleThreshold time.Duration
}

// Decision is the single writer of the session's current status. One
// lives per supervisor; Apply is called sequentially from the decision
// goroutine so there is no concurrent mutation.
type Decision struct {
	cfg DecisionConfig

	current session.AgentStatus
	tool    *string
	work    *string

	lastTranscriptAt     time.Time
	lastPaneStableAt     time.Time
	lastStablePaneHadBox bool
	lastStablePaneSeen   bool
}

// NewDecision seeds state from the config's InitialStatus.
func NewDecision(cfg DecisionConfig) *Decision {
	initial := cfg.InitialStatus
	if initial == "" {
		initial = session.AgentStatusStarting
	}
	return &Decision{cfg: cfg, current: initial}
}

// Current returns the decision machine's current status snapshot. Safe
// only from the decision goroutine.
func (d *Decision) Current() session.AgentStatus { return d.current }

// Tool returns the last known tool reported by the adapter.
func (d *Decision) Tool() *string { return d.tool }

// Work returns the last known work descriptor reported by the adapter.
func (d *Decision) Work() *string { return d.work }

// Apply feeds one Signal into the state machine and returns the resulting
// status along with whether that's a transition from the previous status.
// The rules implement the decision table from the agent-status ticket:
//
//   - Source==liveness                 → ended
//   - Source==transcript && IsError    → error
//   - starting → idle                  on first transcript line OR first stable pane with no box
//   - * → working                      on transcript activity OR pane-hash change with no box match on stable
//   - * → awaiting_input               ONLY when Source==pane && Stable && HasBox
//   - awaiting_input → working         on next transcript line OR stable pane with no box
//   - working → idle                   ONLY when IdleThreshold elapsed AND last pane stable AND last stable had no box
//
// The last conjunction is the Claude permission-dialog bug fix: today's
// tailer trips to idle on transcript silence alone, which paints `idle`
// on the dashboard while a permission box is still on screen.
func (d *Decision) Apply(s Signal) (session.AgentStatus, bool) {
	prev := d.current

	// Once ended, stay ended — late transcript writes from a killed agent
	// (or a race between liveness-disappear and in-flight stdout) must not
	// resurrect the session.
	if d.current == session.AgentStatusEnded {
		return d.current, false
	}

	// Terminal states dominate regardless of current status.
	switch {
	case s.Source == SourceLiveness:
		d.current = session.AgentStatusEnded
	case s.Source == SourceTranscript && s.IsError:
		d.current = session.AgentStatusError
	case s.Source == SourceTranscript:
		d.lastTranscriptAt = s.At
		if s.Tool != nil {
			d.tool = s.Tool
		}
		if s.Work != nil {
			d.work = s.Work
		}
		if s.Status != "" {
			// Adapters that know their state (codex task_started/task_complete,
			// opencode plugin) speak directly. Only promote out of
			// awaiting_input on non-awaiting updates — a plugin emitting
			// "idle" while a box is visible should not override the pane.
			if d.current == session.AgentStatusAwaitingInput && s.Status == session.AgentStatusAwaitingInput {
				d.current = s.Status
			} else if d.current == session.AgentStatusAwaitingInput && s.Status == session.AgentStatusWorking {
				d.current = session.AgentStatusWorking
			} else if d.current == session.AgentStatusAwaitingInput {
				// Ignore idle-ish transcript hints while awaiting input.
			} else {
				d.current = s.Status
			}
		} else if d.current == session.AgentStatusStarting {
			// Default promotion: transcript activity out of starting → idle,
			// we'll go to working only if the adapter says so or via pane.
			d.current = session.AgentStatusIdle
		} else if d.current == session.AgentStatusIdle {
			// Any transcript line in idle is activity — go working.
			d.current = session.AgentStatusWorking
		}
	case s.Source == SourcePane && s.Stable:
		d.lastPaneStableAt = s.At
		d.lastStablePaneSeen = true
		d.lastStablePaneHadBox = s.HasBox
		switch {
		case s.HasBox:
			d.current = session.AgentStatusAwaitingInput
		case d.current == session.AgentStatusStarting:
			d.current = session.AgentStatusIdle
		case d.current == session.AgentStatusAwaitingInput:
			// Box disappeared on a stable frame — back to working. Anchor
			// the silence window to this moment: any prior transcript line
			// from during the dialog shouldn't block the next idle flip.
			d.current = session.AgentStatusWorking
			d.lastTranscriptAt = s.At
		}
	case s.Source == SourcePane && !s.Stable:
		// Pane changed (hash moved) — evidence of activity. Promote to
		// working unless we're currently waiting on the user.
		if d.current != session.AgentStatusAwaitingInput {
			d.current = session.AgentStatusWorking
		}
	case s.Source == SourceTimer:
		// Silence timer: flip to idle only when ALL conditions hold —
		// transcript silent long enough, and a stable pane without a box
		// has been seen more recently than the last transcript line. The
		// conjunction is what stops `idle` from flickering over Claude
		// permission dialogs.
		if d.cfg.IdleThreshold == 0 {
			break
		}
		if d.current != session.AgentStatusWorking {
			break
		}
		silent := !d.lastTranscriptAt.IsZero() && s.At.Sub(d.lastTranscriptAt) >= d.cfg.IdleThreshold
		if !silent {
			break
		}
		if !d.lastStablePaneSeen {
			break
		}
		if d.lastPaneStableAt.Before(d.lastTranscriptAt) {
			// A transcript line arrived after the last stable pane — can't
			// trust the pane to still reflect the current state.
			break
		}
		if d.lastStablePaneHadBox {
			break
		}
		d.current = session.AgentStatusIdle
	}

	return d.current, d.current != prev
}
