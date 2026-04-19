package agent

import (
	"testing"
	"time"

	"github.com/kareemaly/cortex/internal/session"
)

// TestClaudePermissionRegression is the load-bearing bug acceptance test.
// Scenario: Claude writes a tool_use line, then the transcript goes
// silent for 6 seconds (well past the 5s IdleThreshold) because Claude
// is blocked on a permission dialog; meanwhile the pane stabilizes with
// the permission box visible.
//
// Today's Claude tailer flips to idle on transcript silence alone and
// paints "idle" on the dashboard while the box is still up. This test
// feeds the same sequence through the new decision machine + Claude
// pane patterns; the resulting status MUST be awaiting_input, NOT idle.
func TestClaudePermissionRegression(t *testing.T) {
	adapter, ok := Get("claude")
	if !ok {
		t.Fatal("claude adapter not registered")
	}
	decision := NewDecision(DecisionConfig{
		InitialStatus: session.AgentStatusStarting,
		IdleThreshold: adapter.IdleThreshold,
	})
	t0 := time.Now()

	// 1) Claude writes a tool_use line — working.
	if got, _ := decision.Apply(Signal{
		Source: SourceTranscript,
		Status: session.AgentStatusWorking,
		At:     t0,
	}); got != session.AgentStatusWorking {
		t.Fatalf("after transcript line: got %v, want working", got)
	}

	// 2) Pane stabilizes 1s later with the permission box on screen. Feed
	//    the real Claude permission fixture through the adapter's box
	//    patterns so the test also exercises the regex shape.
	rawTail := claudeFixture(t, "awaiting_input_bash")
	_, implied, hitBox := adapter.PanePatterns.MatchFirst(rawTail)
	if !hitBox {
		t.Fatal("pane patterns did not match permission fixture — the regression test can't fire")
	}
	if got, _ := decision.Apply(Signal{
		Source: SourcePane,
		Stable: true,
		HasBox: true,
		Status: implied,
		At:     t0.Add(1 * time.Second),
	}); got != session.AgentStatusAwaitingInput {
		t.Fatalf("after stable-with-box: got %v, want awaiting_input", got)
	}

	// 3) 6s of silence pass — timer fires. The broken code path flips to
	//    idle here; the fixed one must leave us on awaiting_input because
	//    the last stable pane carried a box.
	got, _ := decision.Apply(Signal{
		Source: SourceTimer,
		At:     t0.Add(7 * time.Second),
	})
	if got != session.AgentStatusAwaitingInput {
		t.Fatalf("PERMISSION DIALOG BUG REGRESSED: got %v after silence timer, want awaiting_input", got)
	}
}
