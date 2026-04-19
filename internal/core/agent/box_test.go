package agent

import (
	"regexp"
	"testing"

	"github.com/kareemaly/cortex/internal/session"
)

// fixture patterns mimic Claude's permission dialog shape without
// depending on the real adapter (that lands in Step 5).
func permissionPattern() *BoxPattern {
	return &BoxPattern{
		Name:    "claude_permission_test",
		Border:  regexp.MustCompile(`(?m)^╭───`),
		Anchor:  regexp.MustCompile(`(?m)❯\s+\d+\.\s+(Yes|No)`),
		Implies: session.AgentStatusAwaitingInput,
	}
}

func TestBoxPatternMatchRequiresBothBorderAndAnchor(t *testing.T) {
	p := permissionPattern()
	// Border only — no anchor keyword.
	border := []byte("╭─── something\n│ body │\n╰─── end\n")
	if _, ok := p.Match(border); ok {
		t.Error("border alone must not match")
	}
	// Anchor only — no border.
	anchor := []byte("❯ 1. Yes, I agree\n")
	if _, ok := p.Match(anchor); ok {
		t.Error("anchor alone must not match")
	}
}

func TestBoxPatternMatchHits(t *testing.T) {
	p := permissionPattern()
	raw := []byte("╭────────────────────────────╮\n│ Permission request          │\n│   ❯ 1. Yes                  │\n│     2. No                   │\n╰────────────────────────────╯\n")
	got, ok := p.Match(raw)
	if !ok {
		t.Fatal("expected match")
	}
	if got != session.AgentStatusAwaitingInput {
		t.Errorf("got = %v, want awaiting_input", got)
	}
	if p.Stats().Matches != 1 || p.Stats().Attempts != 1 {
		t.Errorf("counters = %+v, want attempts=1 matches=1", p.Stats())
	}
}

func TestBoxPatternMatchRecordsAttemptsEvenOnMiss(t *testing.T) {
	p := permissionPattern()
	p.Match([]byte("nothing interesting here"))
	p.Match([]byte("still nothing"))
	if p.Stats().Attempts != 2 {
		t.Errorf("attempts = %d, want 2", p.Stats().Attempts)
	}
	if p.Stats().Matches != 0 {
		t.Errorf("matches = %d, want 0", p.Stats().Matches)
	}
}

func TestPanePatternsMatchFirstWinsAndRespectsSearchTail(t *testing.T) {
	first := &BoxPattern{
		Name:    "generic",
		Border:  regexp.MustCompile(`(?m)^╭───`),
		Anchor:  regexp.MustCompile(`permission`),
		Implies: session.AgentStatusAwaitingInput,
	}
	second := permissionPattern()
	pp := PanePatterns{SearchTailLines: 3, Boxes: []*BoxPattern{first, second}}

	// Only the LAST three lines feed into matching — the border in the
	// first two lines should not count.
	raw := []byte(
		"╭──────────\n" +
			"│ something │\n" +
			"scrolled off\n" +
			"line A\n" +
			"line B\n" +
			"no permission here\n",
	)
	if _, _, ok := pp.MatchFirst(raw); ok {
		t.Error("match fired outside SearchTailLines window")
	}
}

func TestCollapseRuns(t *testing.T) {
	got := collapseRuns("─────────end", '─')
	want := "───end"
	if got != want {
		t.Errorf("collapseRuns: got %q, want %q", got, want)
	}
}
