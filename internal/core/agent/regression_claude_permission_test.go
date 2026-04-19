package agent

import (
	"testing"
	"time"

	"github.com/kareemaly/cortex/internal/session"
)

// claudeFixture lives in adapter_claude_test.go in the same package.

// TestClaudePermissionEndToEnd feeds a representative sequence through the
// supervisor's decision machine: working transcript line → stable pane with
// the permission box visible → silent timer ticks. The result MUST stay
// awaiting_input throughout the silence — this is the bug that motivated
// the rewrite (`0d8abcaa`).
func TestClaudePermissionEndToEnd(t *testing.T) {
	adapter, ok := Get("claude")
	if !ok {
		t.Fatal("claude adapter not registered")
	}
	d := NewDecision(DecisionConfig{
		InitialStatus: session.AgentStatusStarting,
		IdleWindow:    adapter.IdleWindow,
	})
	t0 := time.Now()

	// 1) Claude writes a tool_use line → working.
	d.Apply(Signal{Source: SourceTranscript, Activity: true, At: t0})
	if got := d.Current(); got != session.AgentStatusWorking {
		t.Fatalf("after transcript line: %v, want working", got)
	}

	// 2) Pane shows the permission dialog → awaiting_input.
	content := claudeFixture(t, "awaiting_input_bash")
	hasPhrase := adapter.MatchAwaitingInput(content) != ""
	d.Apply(Signal{
		Source:            SourcePane,
		At:                t0.Add(1 * time.Second),
		HasAwaitingPhrase: hasPhrase,
	})
	if got := d.Current(); got != session.AgentStatusAwaitingInput {
		t.Fatalf("after pane with permission box: %v, want awaiting_input", got)
	}

	// 3) Six seconds of silence — pane keeps reporting the dialog (still on
	//    screen). Status MUST stay awaiting_input. The old timer-driven idle
	//    flip is what this test prevents from regressing.
	for i := 1; i <= 12; i++ {
		d.Apply(Signal{
			Source:            SourcePane,
			At:                t0.Add(time.Duration(1+i/2) * time.Second),
			HasAwaitingPhrase: hasPhrase,
		})
	}
	if got := d.Current(); got != session.AgentStatusAwaitingInput {
		t.Fatalf("PERMISSION DIALOG BUG REGRESSED: %v after held silence, want awaiting_input", got)
	}
}

// TestClaudeWorkingDuringSilentToolCall is the new regression for the bug
// that motivated this rewrite: a Claude session running a long silent tool
// call (e.g. `make test-integration`) used to flip working→idle because the
// observer's glyph stripping defeated the spinner-driven hash movement and
// the decision rule treated stable-pane-without-box as evidence of idle.
//
// In the new model, raw pane hashing means the spinner DOES move the hash.
// This test simulates the supervisor seeing pane Changed=true ticks during
// the silent tool execution and asserts working holds.
func TestClaudeWorkingDuringSilentToolCall(t *testing.T) {
	adapter, ok := Get("claude")
	if !ok {
		t.Fatal("claude adapter not registered")
	}
	d := NewDecision(DecisionConfig{
		InitialStatus: session.AgentStatusStarting,
		IdleWindow:    adapter.IdleWindow,
	})
	t0 := time.Now()
	d.Apply(Signal{Source: SourceTranscript, Activity: true, At: t0})

	// 10 seconds of pane ticks where the spinner keeps moving the hash.
	// Transcript is silent the whole time (long tool call).
	for i := 1; i <= 20; i++ {
		d.Apply(Signal{
			Source:  SourcePane,
			Changed: true,
			At:      t0.Add(time.Duration(i) * 500 * time.Millisecond),
		})
	}

	if got := d.Current(); got != session.AgentStatusWorking {
		t.Fatalf("BUG 0d8abcaa REGRESSED: %v during silent tool call, want working", got)
	}
}
