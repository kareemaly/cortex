package observer

import (
	"sync"
	"testing"
	"time"
)

// fakeCapture lets a test script the sequence of pane captures the observer
// sees. Each call consumes the next entry; if there are fewer entries than
// calls, the last entry repeats.
type fakeCapture struct {
	mu    sync.Mutex
	plans map[string][][]byte
}

func (f *fakeCapture) capture(target string) ([]byte, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	plan := f.plans[target]
	if len(plan) == 0 {
		return nil, nil
	}
	next := plan[0]
	if len(plan) > 1 {
		f.plans[target] = plan[1:]
	}
	return next, nil
}

func TestPollEmitsOneSnapshotPerTickPerPane(t *testing.T) {
	fc := &fakeCapture{plans: map[string][][]byte{
		"s:0.0": {[]byte("hello\n")},
	}}
	sink := make(chan Snapshot, 8)
	o := New(Options{Capture: fc.capture})
	o.Register(Pane{SessionID: "sess-1", Target: "s:0.0", Sink: sink})

	base := time.Now()
	for i := range 3 {
		o.poll(base.Add(time.Duration(i) * 100 * time.Millisecond))
	}

	close(sink)
	var snaps []Snapshot
	for s := range sink {
		snaps = append(snaps, s)
	}
	if len(snaps) != 3 {
		t.Fatalf("want 3 snapshots, got %d", len(snaps))
	}
	if snaps[0].Changed {
		t.Errorf("first snapshot must not be Changed (no prior hash)")
	}
	if snaps[1].Changed || snaps[2].Changed {
		t.Errorf("holding content identical should not report Changed")
	}
	if snaps[0].SessionID != "sess-1" {
		t.Errorf("SessionID = %q, want sess-1", snaps[0].SessionID)
	}
}

func TestPollReportsChangedWhenContentMoves(t *testing.T) {
	fc := &fakeCapture{plans: map[string][][]byte{
		"s:0.0": {
			[]byte("first\n"),
			[]byte("first\n"),
			[]byte("second\n"),
			[]byte("second\n"),
		},
	}}
	sink := make(chan Snapshot, 8)
	o := New(Options{Capture: fc.capture})
	o.Register(Pane{SessionID: "s", Target: "s:0.0", Sink: sink})

	base := time.Now()
	for i := range 4 {
		o.poll(base.Add(time.Duration(i) * 100 * time.Millisecond))
	}

	close(sink)
	var snaps []Snapshot
	for s := range sink {
		snaps = append(snaps, s)
	}
	if len(snaps) != 4 {
		t.Fatalf("want 4 snapshots, got %d", len(snaps))
	}
	// Only the transition tick (index 2) reports Changed=true.
	if snaps[0].Changed {
		t.Errorf("snap[0].Changed = true, want false (first seen)")
	}
	if snaps[1].Changed {
		t.Errorf("snap[1].Changed = true, want false (content held)")
	}
	if !snaps[2].Changed {
		t.Errorf("snap[2].Changed = false, want true (content moved)")
	}
	if snaps[3].Changed {
		t.Errorf("snap[3].Changed = true, want false (content held after move)")
	}
}

// TestRawHashDistinguishesSpinnerFrames: the prior implementation stripped
// braille spinner runes before hashing, which turned "agent is alive with a
// rotating spinner" into a stable plateau. The supervisor then flipped
// working→idle while the agent was still working. Raw hashing preserves the
// frame difference, so each spinner rotation reports Changed=true and the
// supervisor keeps the session on working.
func TestRawHashDistinguishesSpinnerFrames(t *testing.T) {
	fc := &fakeCapture{plans: map[string][][]byte{
		"s:0.0": {
			[]byte("busy ⠋\n"),
			[]byte("busy ⠏\n"),
			[]byte("busy ⠙\n"),
		},
	}}
	sink := make(chan Snapshot, 4)
	o := New(Options{Capture: fc.capture})
	o.Register(Pane{SessionID: "s", Target: "s:0.0", Sink: sink})

	base := time.Now()
	for i := range 3 {
		o.poll(base.Add(time.Duration(i) * 100 * time.Millisecond))
	}

	close(sink)
	changes := 0
	for s := range sink {
		if s.Changed {
			changes++
		}
	}
	// Ticks 2 and 3 each differ from their predecessor → two Changed signals.
	if changes != 2 {
		t.Errorf("want 2 Changed snapshots across spinner frames, got %d", changes)
	}
}

func TestRegisterCancelRemovesPane(t *testing.T) {
	fc := func(string) ([]byte, error) { return []byte("x"), nil }
	o := New(Options{Capture: fc})
	cancel := o.Register(Pane{SessionID: "s", Target: "s:0.0", Sink: make(chan Snapshot, 1)})
	cancel()
	cancel() // second call must be a no-op

	o.mu.Lock()
	defer o.mu.Unlock()
	if len(o.panes) != 0 {
		t.Errorf("cancel left pane registered: %d", len(o.panes))
	}
}

func TestMetricsTrackCaptures(t *testing.T) {
	calls := 0
	fc := func(string) ([]byte, error) {
		calls++
		return []byte("x"), nil
	}
	o := New(Options{Capture: fc})
	o.Register(Pane{SessionID: "s", Target: "s:0.0", Sink: make(chan Snapshot, 4)})

	base := time.Now()
	for i := range 3 {
		o.poll(base.Add(time.Duration(i) * 10 * time.Millisecond))
	}

	m := o.Metrics()
	if m.Captures != 3 {
		t.Errorf("captures = %d, want 3", m.Captures)
	}
	if m.CaptureErrors != 0 {
		t.Errorf("capture errors = %d, want 0", m.CaptureErrors)
	}
}

func TestMetricsCountsSinkDrops(t *testing.T) {
	fc := func(string) ([]byte, error) { return []byte("x"), nil }
	sink := make(chan Snapshot) // unbuffered, no reader → every send drops
	o := New(Options{Capture: fc})
	o.Register(Pane{SessionID: "s", Target: "s:0.0", Sink: sink})

	base := time.Now()
	for i := range 3 {
		o.poll(base.Add(time.Duration(i) * 10 * time.Millisecond))
	}

	if got := o.Metrics().SinkDrops; got < 3 {
		t.Errorf("SinkDrops = %d, want >= 3", got)
	}
}
