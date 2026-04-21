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
