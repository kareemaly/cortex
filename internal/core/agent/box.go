package agent

import (
	"bytes"
	"regexp"
	"strings"
	"sync/atomic"

	"github.com/kareemaly/cortex/internal/session"
)

// PanePatterns holds an ordered list of box patterns to try on the tail
// of a pane capture. First match wins; this lets adapters declare
// specific "awaiting_input" box shapes ahead of permissive fallbacks.
//
// SearchTailLines limits how many trailing lines of the pane we consider
// when matching. The observer already caps this on capture (default 12)
// but adapters may narrow further if their TUI has a consistent layout.
type PanePatterns struct {
	SearchTailLines int
	Boxes           []*BoxPattern
}

// BoxPattern describes one message box we recognize on a pane. A match
// requires BOTH the structural border AND the keyword anchor to hit
// somewhere within the same tail window — either alone is rejected. This
// is the key discipline that prevents anchors like "Yes" from firing on
// stray text that happens to contain the word.
//
// Implies is the agent status the match implies; most permission dialogs
// map to AgentStatusAwaitingInput.
//
// attempts/matches are atomic counters exposed on the telemetry debug
// endpoint so silent TUI redesigns (pattern stops firing) can be noticed.
type BoxPattern struct {
	Name    string
	Border  *regexp.Regexp
	Anchor  *regexp.Regexp
	Implies session.AgentStatus

	attempts atomic.Int64
	matches  atomic.Int64
}

// Stats returns a snapshot of the pattern's attempts/matches counters.
type Stats struct {
	Name     string `json:"name"`
	Attempts int64  `json:"attempts"`
	Matches  int64  `json:"matches"`
}

// Stats returns the pattern's current counters.
func (b *BoxPattern) Stats() Stats {
	return Stats{Name: b.Name, Attempts: b.attempts.Load(), Matches: b.matches.Load()}
}

// Match runs the pattern against rawTail. It records the attempt (and the
// match, on success) on atomic counters and returns the Implies status on
// a hit. On a miss it returns ("", false).
func (b *BoxPattern) Match(rawTail []byte) (session.AgentStatus, bool) {
	b.attempts.Add(1)
	if b.Border == nil || b.Anchor == nil {
		return "", false
	}
	normalized := normalizeForBoxMatch(rawTail)
	if !b.Border.Match(normalized) || !b.Anchor.Match(normalized) {
		return "", false
	}
	b.matches.Add(1)
	return b.Implies, true
}

// MatchFirst runs the patterns in order and returns the first that
// matches. Callers use this when they need the box name for telemetry or
// when debugging — most decision-path callers only need a boolean.
func (p PanePatterns) MatchFirst(rawTail []byte) (*BoxPattern, session.AgentStatus, bool) {
	if p.SearchTailLines > 0 {
		rawTail = lastNLines(rawTail, p.SearchTailLines)
	}
	for _, box := range p.Boxes {
		if status, ok := box.Match(rawTail); ok {
			return box, status, true
		}
	}
	return nil, "", false
}

// AllStats returns counters for every pattern registered across every
// adapter. The telemetry endpoint in Step 8 exposes this verbatim.
func AllStats() []Stats {
	out := make([]Stats, 0)
	for _, a := range All() {
		for _, box := range a.PanePatterns.Boxes {
			out = append(out, box.Stats())
		}
	}
	return out
}

// normalizeForBoxMatch is the anti-false-positive scrubber applied before
// regex match: trailing whitespace stripped per line, long runs of `─`
// collapsed to three characters so border regexes match regardless of
// terminal width, ANSI already stripped by tmux capture-pane.
func normalizeForBoxMatch(in []byte) []byte {
	lines := bytes.Split(in, []byte("\n"))
	for i, l := range lines {
		l = bytes.TrimRight(l, " \t\r")
		s := string(l)
		s = collapseRuns(s, '─')
		s = collapseRuns(s, '━')
		s = strings.ReplaceAll(s, "    ", "  ") // normalize long space runs
		lines[i] = []byte(s)
	}
	return bytes.Join(lines, []byte("\n"))
}

// collapseRuns reduces any run of 4+ consecutive `r` runes down to 3
// so regexes written for a terminal width of 40 still match at 80.
func collapseRuns(s string, r rune) string {
	var out strings.Builder
	out.Grow(len(s))
	run := 0
	for _, c := range s {
		if c == r {
			run++
			if run <= 3 {
				out.WriteRune(c)
			}
			continue
		}
		run = 0
		out.WriteRune(c)
	}
	return out.String()
}

// lastNLines mirrors observer.lastNLines but is duplicated here to avoid
// an import cycle (adapter -> observer -> adapter for PanePatterns).
func lastNLines(p []byte, n int) []byte {
	if n <= 0 || len(p) == 0 {
		return p
	}
	end := len(p)
	if p[end-1] == '\n' {
		end--
	}
	count := 0
	for i := end - 1; i >= 0; i-- {
		if p[i] == '\n' {
			count++
			if count == n {
				return p[i+1:]
			}
		}
	}
	return p
}
