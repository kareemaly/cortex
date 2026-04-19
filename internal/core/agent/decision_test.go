package agent

import (
	"testing"
	"time"

	"github.com/kareemaly/cortex/internal/session"
)

func TestDecisionIgnoresIdleTimerWithoutStablePaneEvidence(t *testing.T) {
	d := NewDecision(DecisionConfig{
		InitialStatus: session.AgentStatusStarting,
		IdleThreshold: 5 * time.Second,
	})
	t0 := time.Now()
	// Transcript line → working.
	d.Apply(Signal{Source: SourceTranscript, Status: session.AgentStatusWorking, At: t0})
	// Silence timer fires 10s later, but no pane stable has ever been seen →
	// must NOT flip to idle. This is the core permission-dialog bug fix.
	status, changed := d.Apply(Signal{Source: SourceTimer, At: t0.Add(10 * time.Second)})
	if changed {
		t.Fatalf("transitioned on timer without pane evidence: %v", status)
	}
	if status != session.AgentStatusWorking {
		t.Errorf("current = %v, want working", status)
	}
}

func TestDecisionIdleTimerRequiresBoxFreeStablePane(t *testing.T) {
	d := NewDecision(DecisionConfig{IdleThreshold: 5 * time.Second})
	t0 := time.Now()
	d.Apply(Signal{Source: SourceTranscript, Status: session.AgentStatusWorking, At: t0})
	// Stable pane arrives WITH a box — permission dialog visible.
	d.Apply(Signal{Source: SourcePane, Stable: true, HasBox: true, At: t0.Add(1 * time.Second)})
	// Timer fires at the idle threshold — must NOT drop to idle because the
	// last stable pane carried a box.
	got, _ := d.Apply(Signal{Source: SourceTimer, At: t0.Add(10 * time.Second)})
	if got == session.AgentStatusIdle {
		t.Errorf("flipped to idle while pane box visible")
	}
}

func TestDecisionIdleTimerFiresWithBoxFreeStablePane(t *testing.T) {
	d := NewDecision(DecisionConfig{IdleThreshold: 5 * time.Second})
	t0 := time.Now()
	d.Apply(Signal{Source: SourceTranscript, Status: session.AgentStatusWorking, At: t0})
	// Stable pane without a box (agent is just waiting silently).
	d.Apply(Signal{Source: SourcePane, Stable: true, HasBox: false, At: t0.Add(1 * time.Second)})
	got, _ := d.Apply(Signal{Source: SourceTimer, At: t0.Add(10 * time.Second)})
	if got != session.AgentStatusIdle {
		t.Errorf("got = %v, want idle", got)
	}
}

func TestDecisionPaneBoxForcesAwaitingInput(t *testing.T) {
	d := NewDecision(DecisionConfig{})
	t0 := time.Now()
	d.Apply(Signal{Source: SourceTranscript, Status: session.AgentStatusWorking, At: t0})
	got, changed := d.Apply(Signal{Source: SourcePane, Stable: true, HasBox: true, At: t0.Add(1 * time.Second)})
	if !changed {
		t.Fatal("expected transition to awaiting_input")
	}
	if got != session.AgentStatusAwaitingInput {
		t.Errorf("got = %v, want awaiting_input", got)
	}
}

func TestDecisionAwaitingInputReturnsToWorkingOnBoxClear(t *testing.T) {
	d := NewDecision(DecisionConfig{})
	t0 := time.Now()
	d.Apply(Signal{Source: SourceTranscript, Status: session.AgentStatusWorking, At: t0})
	d.Apply(Signal{Source: SourcePane, Stable: true, HasBox: true, At: t0.Add(1 * time.Second)})
	got, _ := d.Apply(Signal{Source: SourcePane, Stable: true, HasBox: false, At: t0.Add(2 * time.Second)})
	if got != session.AgentStatusWorking {
		t.Errorf("got = %v, want working", got)
	}
}

func TestDecisionLivenessEndsSession(t *testing.T) {
	d := NewDecision(DecisionConfig{InitialStatus: session.AgentStatusWorking})
	got, _ := d.Apply(Signal{Source: SourceLiveness, At: time.Now()})
	if got != session.AgentStatusEnded {
		t.Errorf("got = %v, want ended", got)
	}
}

func TestDecisionTranscriptErrorFlipsToError(t *testing.T) {
	d := NewDecision(DecisionConfig{InitialStatus: session.AgentStatusWorking})
	got, _ := d.Apply(Signal{Source: SourceTranscript, IsError: true, At: time.Now()})
	if got != session.AgentStatusError {
		t.Errorf("got = %v, want error", got)
	}
}

func TestDecisionTranscriptStatusIgnoredWhileAwaitingInput(t *testing.T) {
	// A stale "idle" from the plugin must not override a visible box.
	d := NewDecision(DecisionConfig{})
	t0 := time.Now()
	d.Apply(Signal{Source: SourceTranscript, Status: session.AgentStatusWorking, At: t0})
	d.Apply(Signal{Source: SourcePane, Stable: true, HasBox: true, At: t0.Add(1 * time.Second)})
	got, _ := d.Apply(Signal{Source: SourceTranscript, Status: session.AgentStatusIdle, At: t0.Add(2 * time.Second)})
	if got != session.AgentStatusAwaitingInput {
		t.Errorf("got = %v, want awaiting_input (idle ignored while box visible)", got)
	}
}

func TestDecisionStartingPromotesOnStablePane(t *testing.T) {
	d := NewDecision(DecisionConfig{InitialStatus: session.AgentStatusStarting})
	got, _ := d.Apply(Signal{Source: SourcePane, Stable: true, HasBox: false, At: time.Now()})
	if got != session.AgentStatusIdle {
		t.Errorf("got = %v, want idle", got)
	}
}

func TestDecisionStartingPromotesOnTranscript(t *testing.T) {
	d := NewDecision(DecisionConfig{InitialStatus: session.AgentStatusStarting})
	got, _ := d.Apply(Signal{Source: SourceTranscript, At: time.Now()})
	if got != session.AgentStatusIdle {
		t.Errorf("got = %v, want idle", got)
	}
}

func TestDecisionPaneChangeWhileIdlePromotesToWorking(t *testing.T) {
	d := NewDecision(DecisionConfig{InitialStatus: session.AgentStatusIdle})
	got, _ := d.Apply(Signal{Source: SourcePane, Stable: false, At: time.Now()})
	if got != session.AgentStatusWorking {
		t.Errorf("got = %v, want working", got)
	}
}
