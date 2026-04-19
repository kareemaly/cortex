package agent

import (
	"testing"
	"time"

	"github.com/kareemaly/cortex/internal/session"
)

// The decision machine is a precedence table. These tests cover one row of
// the table per case and one regression for each historic foot-gun.

func TestLivenessGoneBecomesEnded(t *testing.T) {
	d := NewDecision(DecisionConfig{InitialStatus: session.AgentStatusWorking})
	got, changed := d.Apply(Signal{Source: SourceLiveness, At: time.Now()})
	if !changed || got != session.AgentStatusEnded {
		t.Errorf("got = %v changed=%v, want ended/true", got, changed)
	}
}

func TestEndedIsTerminal(t *testing.T) {
	// Late transcript writes from a killed agent must not resurrect.
	d := NewDecision(DecisionConfig{InitialStatus: session.AgentStatusWorking})
	d.Apply(Signal{Source: SourceLiveness, At: time.Now()})
	got, changed := d.Apply(Signal{Source: SourceTranscript, Activity: true, At: time.Now()})
	if changed || got != session.AgentStatusEnded {
		t.Errorf("ended must be sticky; got = %v changed=%v", got, changed)
	}
}

func TestTranscriptErrorBecomesError(t *testing.T) {
	d := NewDecision(DecisionConfig{InitialStatus: session.AgentStatusWorking})
	got, changed := d.Apply(Signal{Source: SourceTranscript, IsError: true, At: time.Now()})
	if !changed || got != session.AgentStatusError {
		t.Errorf("got = %v changed=%v, want error/true", got, changed)
	}
}

func TestAuthoritativeTranscriptStatusOverrides(t *testing.T) {
	// OpenCode plugin / Codex task events push explicit status; that wins.
	d := NewDecision(DecisionConfig{InitialStatus: session.AgentStatusStarting})
	got, _ := d.Apply(Signal{
		Source: SourceTranscript, Status: session.AgentStatusWorking, At: time.Now(),
	})
	if got != session.AgentStatusWorking {
		t.Errorf("got = %v, want working from authoritative status", got)
	}
}

func TestPanePhraseMatchBecomesAwaitingInput(t *testing.T) {
	d := NewDecision(DecisionConfig{InitialStatus: session.AgentStatusWorking})
	got, changed := d.Apply(Signal{
		Source: SourcePane, HasAwaitingPhrase: true, At: time.Now(),
	})
	if !changed || got != session.AgentStatusAwaitingInput {
		t.Errorf("got = %v changed=%v, want awaiting_input/true", got, changed)
	}
}

func TestPaneChangePromotesIdleToWorking(t *testing.T) {
	d := NewDecision(DecisionConfig{InitialStatus: session.AgentStatusIdle})
	got, _ := d.Apply(Signal{Source: SourcePane, Changed: true, At: time.Now()})
	if got != session.AgentStatusWorking {
		t.Errorf("got = %v, want working", got)
	}
}

func TestPaneChangeClearsAwaitingInput(t *testing.T) {
	// Dialog dismissed → pane moves → back to working, even though no phrase.
	d := NewDecision(DecisionConfig{InitialStatus: session.AgentStatusAwaitingInput})
	got, _ := d.Apply(Signal{Source: SourcePane, Changed: true, At: time.Now()})
	if got != session.AgentStatusWorking {
		t.Errorf("got = %v, want working", got)
	}
}

func TestTranscriptActivityHoldsAwaitingInput(t *testing.T) {
	// Plugin/agent might emit a non-status transcript line while the user is
	// still blocking — don't lose awaiting_input over it.
	d := NewDecision(DecisionConfig{InitialStatus: session.AgentStatusAwaitingInput})
	got, _ := d.Apply(Signal{Source: SourceTranscript, Activity: true, At: time.Now()})
	if got != session.AgentStatusAwaitingInput {
		t.Errorf("got = %v, want awaiting_input held", got)
	}
}

func TestStaleIdleStatusIgnoredWhileAwaitingInput(t *testing.T) {
	// Carve-out: an authoritative "idle" transcript event that arrives while
	// a pane phrase has put us into awaiting_input must not undo the flip.
	d := NewDecision(DecisionConfig{InitialStatus: session.AgentStatusAwaitingInput})
	got, changed := d.Apply(Signal{
		Source: SourceTranscript, Status: session.AgentStatusIdle, At: time.Now(),
	})
	if changed || got != session.AgentStatusAwaitingInput {
		t.Errorf("got = %v changed=%v, want awaiting_input held", got, changed)
	}
}

func TestIdleWindowOnlyAppliesToWorking(t *testing.T) {
	d := NewDecision(DecisionConfig{
		InitialStatus: session.AgentStatusStarting,
		IdleWindow:    1 * time.Second,
	})
	t0 := time.Now()
	got, _ := d.Apply(Signal{Source: SourcePane, At: t0.Add(10 * time.Second)})
	if got != session.AgentStatusStarting {
		t.Errorf("starting must not decay to idle; got %v", got)
	}
}

func TestIdleWindowFiresAfterQuiet(t *testing.T) {
	d := NewDecision(DecisionConfig{IdleWindow: 1 * time.Second})
	t0 := time.Now()
	d.Apply(Signal{Source: SourceTranscript, Activity: true, At: t0})
	// Quiet pane tick well past the window → idle.
	got, _ := d.Apply(Signal{Source: SourcePane, At: t0.Add(2 * time.Second)})
	if got != session.AgentStatusIdle {
		t.Errorf("got = %v, want idle after quiet window", got)
	}
}

func TestActivityResetsIdleClock(t *testing.T) {
	// New transcript activity must push the idle deadline back.
	d := NewDecision(DecisionConfig{IdleWindow: 1 * time.Second})
	t0 := time.Now()
	d.Apply(Signal{Source: SourceTranscript, Activity: true, At: t0})
	d.Apply(Signal{Source: SourceTranscript, Activity: true, At: t0.Add(800 * time.Millisecond)})
	// Pane tick at t0+1.5s — past the original deadline but within IdleWindow
	// of the second activity. Must not flip.
	got, _ := d.Apply(Signal{Source: SourcePane, At: t0.Add(1500 * time.Millisecond)})
	if got != session.AgentStatusWorking {
		t.Errorf("got = %v, want working (activity reset clock)", got)
	}
}

func TestPaneChangeKeepsWorkingFromDecaying(t *testing.T) {
	// Bug 0d8abcaa root-cause regression: pane that keeps changing (live
	// counter, scrolling output, spinner glyph) should never decay to idle.
	d := NewDecision(DecisionConfig{IdleWindow: 1 * time.Second})
	t0 := time.Now()
	d.Apply(Signal{Source: SourceTranscript, Activity: true, At: t0})
	// Pane keeps moving every tick.
	for i := 1; i < 6; i++ {
		d.Apply(Signal{Source: SourcePane, Changed: true, At: t0.Add(time.Duration(i) * 500 * time.Millisecond)})
	}
	if got := d.Current(); got != session.AgentStatusWorking {
		t.Errorf("got = %v, want working held by pane changes", got)
	}
}

func TestNoIdleWindowMeansNoTimeoutDecay(t *testing.T) {
	// Adapters with IdleWindow=0 stay on working until something else moves
	// them.
	d := NewDecision(DecisionConfig{IdleWindow: 0})
	t0 := time.Now()
	d.Apply(Signal{Source: SourceTranscript, Activity: true, At: t0})
	got, _ := d.Apply(Signal{Source: SourcePane, At: t0.Add(time.Hour)})
	if got != session.AgentStatusWorking {
		t.Errorf("got = %v, want working held with IdleWindow=0", got)
	}
}

func TestTranscriptToolWorkPropagated(t *testing.T) {
	d := NewDecision(DecisionConfig{})
	tool, work := "Bash", "running tests"
	d.Apply(Signal{
		Source:   SourceTranscript,
		Activity: true,
		Tool:     &tool,
		Work:     &work,
		At:       time.Now(),
	})
	if got := d.Tool(); got == nil || *got != "Bash" {
		t.Errorf("Tool not propagated; got %v", got)
	}
	if got := d.Work(); got == nil || *got != "running tests" {
		t.Errorf("Work not propagated; got %v", got)
	}
}
