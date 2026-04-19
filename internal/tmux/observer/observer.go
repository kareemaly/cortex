// Package observer runs a single shared goroutine that polls tmux pane
// contents and publishes stability-detected snapshots to interested
// subscribers. It exists so per-session supervisors in internal/core/agent
// can fold pane observation into their status-decision logic without each
// session spinning its own ticker and tmux command.
//
// Scope (per the agent-status ticket): supervised panes only. The Register
// API is shaped so a future cortex-wide pane observer can enumerate every
// tmux pane, but this observer does not enumerate unsupervised panes.
package observer

import (
	"bytes"
	"crypto/sha256"
	"sort"
	"sync"
	"sync/atomic"
	"time"
	"unicode/utf8"
)

// Defaults per the agent-status ticket refinements:
//
//   - Fast tier 250 ms base tick, matching the hard per-pane floor.
//   - Slow tier 1200 ms when every registered pane reports idle/awaiting_input.
//   - Stability window 400 ms — once a pane's SHA-256 hash has not moved for
//     this long, exactly one Stable=true signal fires per plateau.
//
// Earlier drafts used 400 ms/600 ms but that sits right on the ~1 s
// awaiting-input acceptance boundary; 250/400 gives headroom at negligible
// CPU cost on top of what tmux capture already spends.
const (
	DefaultFastTick        = 250 * time.Millisecond
	DefaultSlowTick        = 1200 * time.Millisecond
	DefaultStabilityWindow = 400 * time.Millisecond
	DefaultSearchTailLines = 12
	DefaultPerPaneFloor    = 250 * time.Millisecond
)

// Signal is what the observer pushes to a pane's Sink channel.
// Stable=true signals fire exactly once per stable plateau.
type Signal struct {
	SessionID string
	Target    string
	Hash      [sha256.Size]byte
	RawTail   []byte
	Stable    bool
	At        time.Time
}

// StatusFn returns the pane's currently-reported status so the observer
// can pick the right tier (fast vs slow) when no pane is actively working.
// Return "working" or "starting" to keep the fast tier running.
type StatusFn func() string

// Pane is the unit of registration. Sink receives every emitted Signal for
// this pane; the caller should buffer it (cap ≥ 1 is enough) so the
// observer's single goroutine never blocks.
type Pane struct {
	SessionID string
	Target    string
	StatusFn  StatusFn
	Sink      chan<- Signal
}

// Capture pulls the current visible pane content. Real callers pass
// tmux.Manager.CapturePane; tests inject a deterministic fake.
type Capture func(target string) ([]byte, error)

// Observer polls registered panes and emits pane signals. One Observer per
// daemon is sufficient; create via New.
type Observer struct {
	capture     Capture
	fastTick    time.Duration
	slowTick    time.Duration
	stability   time.Duration
	searchTail  int
	perPaneHold time.Duration

	mu        sync.Mutex
	panes     map[string]*paneState // keyed by Target
	stopCh    chan struct{}
	stopOnce  sync.Once
	startOnce sync.Once
	started   bool

	// Observability — exposed via Metrics() for the /agent/status/debug
	// endpoint built in Step 8.
	metricCaptures       atomic.Int64
	metricCaptureErrors  atomic.Int64
	metricDurationNanoTS atomic.Int64 // sum of capture durations (ns)
	metricSinkDrops      atomic.Int64 // stability signals dropped because sink was full
}

type paneState struct {
	pane        Pane
	lastHash    [sha256.Size]byte
	lastChange  time.Time
	emittedFor  [sha256.Size]byte // hash whose stability signal has already fired
	stableKnown bool
	lastPolled  time.Time
	everPolled  bool
}

// Options tweaks an Observer's timing. Zero values select defaults; to
// disable the per-pane floor in tests, use DisablePerPaneFloor.
type Options struct {
	FastTick        time.Duration
	SlowTick        time.Duration
	StabilityWindow time.Duration
	SearchTailLines int
	PerPaneFloor    time.Duration
	Capture         Capture
}

// DisablePerPaneFloor explicitly disables the 250 ms per-pane capture floor.
// Intended for tests that drive poll directly with a synthetic clock.
const DisablePerPaneFloor = -time.Nanosecond

// New constructs an Observer but does not start the poll loop. Callers must
// invoke Start to kick off polling and Stop to tear it down.
func New(opts Options) *Observer {
	if opts.Capture == nil {
		panic("observer: Options.Capture is required")
	}
	o := &Observer{
		capture:     opts.Capture,
		fastTick:    opts.FastTick,
		slowTick:    opts.SlowTick,
		stability:   opts.StabilityWindow,
		searchTail:  opts.SearchTailLines,
		perPaneHold: opts.PerPaneFloor,
		panes:       make(map[string]*paneState),
		stopCh:      make(chan struct{}),
	}
	if o.fastTick == 0 {
		o.fastTick = DefaultFastTick
	}
	if o.slowTick == 0 {
		o.slowTick = DefaultSlowTick
	}
	if o.stability == 0 {
		o.stability = DefaultStabilityWindow
	}
	if o.searchTail == 0 {
		o.searchTail = DefaultSearchTailLines
	}
	switch {
	case o.perPaneHold == 0:
		o.perPaneHold = DefaultPerPaneFloor
	case o.perPaneHold < 0:
		o.perPaneHold = 0
	}
	return o
}

// Register adds a pane to the rotation. The returned cancel function
// removes it; cancel is safe to call multiple times. Registering a target
// that already exists replaces the prior Pane (last-writer-wins) and
// returns a cancel tied to the new registration.
func (o *Observer) Register(p Pane) func() {
	if p.Target == "" {
		panic("observer: Pane.Target is required")
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	o.panes[p.Target] = &paneState{pane: p}
	target := p.Target
	// Capture the identity of this particular registration so a late
	// cancel doesn't delete a subsequently-registered pane on the same
	// target.
	registered := o.panes[target]
	return func() {
		o.mu.Lock()
		if cur, ok := o.panes[target]; ok && cur == registered {
			delete(o.panes, target)
		}
		o.mu.Unlock()
	}
}

// Start kicks off the polling goroutine. Safe to call multiple times;
// subsequent calls are no-ops.
func (o *Observer) Start() {
	o.startOnce.Do(func() {
		o.mu.Lock()
		o.started = true
		o.mu.Unlock()
		go o.loop()
	})
}

// Stop halts the polling goroutine. Safe to call once; further calls are
// no-ops.
func (o *Observer) Stop() {
	o.stopOnce.Do(func() { close(o.stopCh) })
}

// Metrics returns a snapshot of observer counters. It is safe to call from
// any goroutine.
func (o *Observer) Metrics() Metrics {
	captures := o.metricCaptures.Load()
	var avg time.Duration
	if captures > 0 {
		avg = time.Duration(o.metricDurationNanoTS.Load() / captures)
	}
	return Metrics{
		Captures:              captures,
		CaptureErrors:         o.metricCaptureErrors.Load(),
		SinkDrops:             o.metricSinkDrops.Load(),
		AvgCaptureDurationMs:  float64(avg) / float64(time.Millisecond),
		TotalCaptureDurations: time.Duration(o.metricDurationNanoTS.Load()),
	}
}

// Metrics is the observability payload. Exposed on /agent/status/debug so
// the "<5% of one core at 20 sessions" claim in the design can actually be
// verified in production.
type Metrics struct {
	Captures              int64         `json:"captures"`
	CaptureErrors         int64         `json:"capture_errors"`
	SinkDrops             int64         `json:"sink_drops"`
	AvgCaptureDurationMs  float64       `json:"avg_capture_duration_ms"`
	TotalCaptureDurations time.Duration `json:"total_capture_duration_ns"`
}

func (o *Observer) loop() {
	for {
		start := time.Now()
		tick := o.poll(start)
		select {
		case <-o.stopCh:
			return
		case <-time.After(tick):
		}
	}
}

// poll captures every registered pane at most once per base tick and
// returns the delay until the next tick. Split out from loop for test
// control: tests drive poll directly with a synthetic clock.
func (o *Observer) poll(now time.Time) time.Duration {
	// Snapshot panes under the lock so capture() can run without holding it.
	o.mu.Lock()
	targets := make([]*paneState, 0, len(o.panes))
	for _, st := range o.panes {
		targets = append(targets, st)
	}
	o.mu.Unlock()

	// Stable iteration order keeps test timing predictable.
	sort.Slice(targets, func(i, j int) bool {
		return targets[i].pane.Target < targets[j].pane.Target
	})

	anyActive := false
	for _, st := range targets {
		// Hard per-pane floor — never capture the same pane more frequently
		// than this regardless of tier. Matters when the slow tier races.
		if st.everPolled && now.Sub(st.lastPolled) < o.perPaneHold {
			continue
		}
		st.lastPolled = now
		st.everPolled = true

		captureStart := time.Now()
		content, err := o.capture(st.pane.Target)
		o.metricCaptures.Add(1)
		o.metricDurationNanoTS.Add(int64(time.Since(captureStart)))
		if err != nil {
			o.metricCaptureErrors.Add(1)
			continue
		}

		// Normalize content before hashing so rotating spinner runes don't
		// reset the plateau on every frame. Agents like Claude render
		// braille-block spinners that otherwise make stability impossible
		// at fast cadence.
		hashInput := stripAnimatedGlyphs(content)
		hash := sha256.Sum256(hashInput)
		changed := hash != st.lastHash
		if changed {
			st.lastHash = hash
			st.lastChange = now
			st.stableKnown = false
			st.emittedFor = [sha256.Size]byte{}
		}

		// Deliver one Stable signal per plateau once the stability window
		// has elapsed with no hash change.
		if !changed && !st.stableKnown && now.Sub(st.lastChange) >= o.stability && hash != st.emittedFor {
			st.emittedFor = hash
			st.stableKnown = true
			signal := Signal{
				SessionID: st.pane.SessionID,
				Target:    st.pane.Target,
				Hash:      hash,
				RawTail:   lastNLines(content, o.searchTail),
				Stable:    true,
				At:        now,
			}
			select {
			case st.pane.Sink <- signal:
			default:
				// Sink is full — drop rather than block the shared loop. The
				// subscriber can recover on the next plateau. SinkDrops is
				// exposed via Metrics so a persistently-full sink becomes
				// visible on /agent/status/debug instead of silent.
				o.metricSinkDrops.Add(1)
			}
		}

		if st.pane.StatusFn != nil {
			switch st.pane.StatusFn() {
			case "working", "starting":
				anyActive = true
			}
		} else {
			anyActive = true
		}
	}

	if anyActive {
		return o.fastTick
	}
	return o.slowTick
}

// stripAnimatedGlyphs removes runes that typically rotate each frame
// (braille spinners, pulse blocks, asterisk spinners), so the SHA-256
// stability check isn't defeated by cosmetic animation. Ported from
// agent-deck's StripSpinnerRunes pattern — critical for Claude which
// renders U+2800–U+28FF braille spinners in its status line.
//
// All stripped runes are 3-byte UTF-8 sequences starting with 0xE2, so
// the fast path short-circuits when that prefix byte is absent — the
// typical terminal frame for most agents hits this path and avoids the
// allocation entirely.
func stripAnimatedGlyphs(p []byte) []byte {
	if len(p) == 0 {
		return p
	}
	if bytes.IndexByte(p, 0xE2) == -1 {
		return p
	}
	out := make([]byte, 0, len(p))
	for i := 0; i < len(p); {
		r, size := utf8.DecodeRune(p[i:])
		skip := false
		switch {
		case r >= 0x2800 && r <= 0x28FF: // braille patterns
			skip = true
		case r == '█' || r == '▓' || r == '▒' || r == '░': // pulse blocks
			skip = true
		case r == '✳' || r == '✽' || r == '✶' || r == '✻' || r == '✢': // claude status glyphs
			skip = true
		}
		if !skip {
			out = append(out, p[i:i+size]...)
		}
		i += size
	}
	return out
}

// lastNLines returns the last n lines of p. Trailing `\n` terminators are
// preserved; an entirely empty trailing line is not counted as an extra
// line so "a\nb\nc\n" with n=2 yields "b\nc\n" (the final two lines) — the
// intuition callers actually want for pane-tail snippets.
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
