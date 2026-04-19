// Package observer runs a single shared goroutine that polls registered
// tmux panes and emits a Snapshot per tick per pane. It is deliberately
// stupid: raw capture, single tick interval, hash the content as-is. The
// supervisor decides what a snapshot means.
//
// The prior design stripped animated runes (braille spinners, pulse
// blocks) before hashing, on the theory that a rotating glyph shouldn't
// reset a "stability plateau." In practice that was exactly backwards:
// the spinner is the agent saying "I'm alive." Stripping it turned silent
// tool execution into a false-idle plateau that flipped working→idle.
// This implementation hashes raw bytes and lets the supervisor trust
// spinner-driven hash movement as proof of liveness.
package observer

import (
	"crypto/sha256"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

// DefaultTick is the fixed poll interval. Single tier — no adaptive fast/slow
// tracks. 500 ms matches claude-squad's interval and is well under any realistic
// dashboard latency budget.
const DefaultTick = 500 * time.Millisecond

// Snapshot is what the observer publishes per pane per tick.
//
// Content holds the raw tmux capture-pane output (ANSI already stripped by
// capture-pane). Subscribers that need to substring-match dialog phrases read
// Content directly. Changed reports whether Hash moved since the prior tick —
// a cheap "agent did something" signal. Changed is always false on the first
// snapshot for a pane.
type Snapshot struct {
	SessionID string
	Target    string
	Hash      [sha256.Size]byte
	Content   []byte
	Changed   bool
	At        time.Time
}

// Pane is the unit of registration. Sink receives every tick's snapshot;
// callers size the buffer for their latency tolerance. Full sinks drop the
// snapshot (metrics.SinkDrops tracks it) rather than blocking the shared
// poll loop.
type Pane struct {
	SessionID string
	Target    string
	Sink      chan<- Snapshot
}

// Capture pulls the current visible pane content. Real callers pass
// tmux.Manager.CapturePane; tests inject a deterministic fake.
type Capture func(target string) ([]byte, error)

// Options tweaks an Observer. Zero values pick defaults.
type Options struct {
	Tick    time.Duration
	Capture Capture
}

// Observer polls registered panes. One per daemon is sufficient; create with
// New, start with Start, stop with Stop.
type Observer struct {
	capture Capture
	tick    time.Duration

	mu        sync.Mutex
	panes     map[string]*paneState
	stopCh    chan struct{}
	stopOnce  sync.Once
	startOnce sync.Once

	metricCaptures     atomic.Int64
	metricCaptureErrs  atomic.Int64
	metricSinkDrops    atomic.Int64
	metricDurationNano atomic.Int64
}

type paneState struct {
	pane     Pane
	lastHash [sha256.Size]byte
	seen     bool
}

// New constructs an Observer. Start must be called separately to begin
// polling.
func New(opts Options) *Observer {
	if opts.Capture == nil {
		panic("observer: Options.Capture is required")
	}
	if opts.Tick == 0 {
		opts.Tick = DefaultTick
	}
	return &Observer{
		capture: opts.Capture,
		tick:    opts.Tick,
		panes:   make(map[string]*paneState),
		stopCh:  make(chan struct{}),
	}
}

// Register adds a pane to the rotation. The returned cancel function removes
// it; safe to call multiple times. Registering the same Target twice replaces
// the prior Pane (last-writer-wins); a cancel returned from the first
// registration is inert against the replacement.
func (o *Observer) Register(p Pane) func() {
	if p.Target == "" {
		panic("observer: Pane.Target is required")
	}
	o.mu.Lock()
	defer o.mu.Unlock()
	o.panes[p.Target] = &paneState{pane: p}
	target := p.Target
	registered := o.panes[target]
	return func() {
		o.mu.Lock()
		if cur, ok := o.panes[target]; ok && cur == registered {
			delete(o.panes, target)
		}
		o.mu.Unlock()
	}
}

// Start kicks off the polling goroutine. Idempotent.
func (o *Observer) Start() {
	o.startOnce.Do(func() { go o.loop() })
}

// Stop halts the polling goroutine. Idempotent.
func (o *Observer) Stop() { o.stopOnce.Do(func() { close(o.stopCh) }) }

func (o *Observer) loop() {
	t := time.NewTicker(o.tick)
	defer t.Stop()
	for {
		select {
		case <-o.stopCh:
			return
		case now := <-t.C:
			o.poll(now)
		}
	}
}

// poll captures every registered pane once and emits one Snapshot per pane.
// Split out from loop for test control — tests drive poll directly with a
// synthetic clock.
func (o *Observer) poll(now time.Time) {
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

	for _, st := range targets {
		captureStart := time.Now()
		content, err := o.capture(st.pane.Target)
		o.metricCaptures.Add(1)
		o.metricDurationNano.Add(int64(time.Since(captureStart)))
		if err != nil {
			o.metricCaptureErrs.Add(1)
			continue
		}

		hash := sha256.Sum256(content)
		changed := false
		if st.seen {
			changed = hash != st.lastHash
		}
		st.lastHash = hash
		st.seen = true

		snap := Snapshot{
			SessionID: st.pane.SessionID,
			Target:    st.pane.Target,
			Hash:      hash,
			Content:   content,
			Changed:   changed,
			At:        now,
		}
		select {
		case st.pane.Sink <- snap:
		default:
			o.metricSinkDrops.Add(1)
		}
	}
}

// Metrics returns a snapshot of observer counters. Safe from any goroutine.
func (o *Observer) Metrics() Metrics {
	captures := o.metricCaptures.Load()
	var avg time.Duration
	if captures > 0 {
		avg = time.Duration(o.metricDurationNano.Load() / captures)
	}
	return Metrics{
		Captures:             captures,
		CaptureErrors:        o.metricCaptureErrs.Load(),
		SinkDrops:            o.metricSinkDrops.Load(),
		AvgCaptureDurationMs: float64(avg) / float64(time.Millisecond),
	}
}

// Metrics is the observability payload exposed by /agent/status/debug so the
// capture-duration claim can be checked in production.
type Metrics struct {
	Captures             int64   `json:"captures"`
	CaptureErrors        int64   `json:"capture_errors"`
	SinkDrops            int64   `json:"sink_drops"`
	AvgCaptureDurationMs float64 `json:"avg_capture_duration_ms"`
}
