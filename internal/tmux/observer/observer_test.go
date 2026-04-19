package observer

import (
	"sync"
	"testing"
	"time"
)

// fakeCapture lets a test script the sequence of pane captures the
// observer sees. Each call consumes the next entry; if there are fewer
// entries than calls, the last entry repeats.
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

func TestPollEmitsOneStableSignalPerPlateau(t *testing.T) {
	fc := &fakeCapture{plans: map[string][][]byte{
		"s:0.0": {[]byte("hello\nworld\n")},
	}}
	sink := make(chan Signal, 4)
	o := New(Options{
		Capture:         fc.capture,
		StabilityWindow: 100 * time.Millisecond,
		PerPaneFloor:    DisablePerPaneFloor,
	})
	o.Register(Pane{
		SessionID: "sess-1",
		Target:    "s:0.0",
		StatusFn:  func() string { return "idle" },
		Sink:      sink,
	})

	base := time.Now()
	// Five polls across 300 ms — first poll sets the hash, subsequent polls
	// find it unchanged; one stable signal should fire once the window
	// elapses. No additional stable signals should follow for the same hash.
	for i := range 5 {
		o.poll(base.Add(time.Duration(i) * 80 * time.Millisecond))
	}

	close(sink)
	var signals []Signal
	for s := range sink {
		signals = append(signals, s)
	}
	if len(signals) != 1 {
		t.Fatalf("expected 1 stable signal, got %d", len(signals))
	}
	if !signals[0].Stable {
		t.Error("expected Stable=true")
	}
	if signals[0].SessionID != "sess-1" {
		t.Errorf("SessionID = %q, want sess-1", signals[0].SessionID)
	}
}

func TestPollResetsPlateauOnContentChange(t *testing.T) {
	fc := &fakeCapture{plans: map[string][][]byte{
		"s:0.0": {
			[]byte("first\n"),
			[]byte("first\n"),
			[]byte("first\n"),
			[]byte("second\n"),
			[]byte("second\n"),
			[]byte("second\n"),
			[]byte("second\n"),
		},
	}}
	sink := make(chan Signal, 8)
	o := New(Options{
		Capture:         fc.capture,
		StabilityWindow: 100 * time.Millisecond,
		PerPaneFloor:    DisablePerPaneFloor,
	})
	o.Register(Pane{SessionID: "s", Target: "s:0.0", Sink: sink})

	base := time.Now()
	for i := range 7 {
		o.poll(base.Add(time.Duration(i) * 80 * time.Millisecond))
	}

	close(sink)
	var stables []Signal
	for s := range sink {
		if s.Stable {
			stables = append(stables, s)
		}
	}
	if len(stables) != 2 {
		t.Fatalf("expected 2 stable signals (one per plateau), got %d", len(stables))
	}
}

func TestPollRespectsPerPaneFloor(t *testing.T) {
	calls := 0
	fc := func(string) ([]byte, error) {
		calls++
		return []byte("x"), nil
	}
	o := New(Options{
		Capture:      fc,
		PerPaneFloor: 200 * time.Millisecond,
	})
	o.Register(Pane{SessionID: "s", Target: "s:0.0", Sink: make(chan Signal, 1)})

	base := time.Now()
	// Two polls 50 ms apart — second must be skipped because of floor.
	o.poll(base)
	o.poll(base.Add(50 * time.Millisecond))
	if calls != 1 {
		t.Errorf("per-pane floor not enforced: calls=%d, want 1", calls)
	}

	// Third poll is past the floor.
	o.poll(base.Add(260 * time.Millisecond))
	if calls != 2 {
		t.Errorf("poll after floor expired: calls=%d, want 2", calls)
	}
}

func TestPollSelectsSlowTierWhenAllPanesIdle(t *testing.T) {
	fc := func(string) ([]byte, error) { return []byte("stable"), nil }
	o := New(Options{
		Capture:      fc,
		FastTick:     50 * time.Millisecond,
		SlowTick:     1 * time.Second,
		PerPaneFloor: DisablePerPaneFloor,
	})
	o.Register(Pane{
		SessionID: "s",
		Target:    "s:0.0",
		StatusFn:  func() string { return "idle" },
		Sink:      make(chan Signal, 1),
	})
	next := o.poll(time.Now())
	if next < 500*time.Millisecond {
		t.Errorf("expected slow tier, got %v", next)
	}
}

func TestPollSelectsFastTierWhenAPaneIsWorking(t *testing.T) {
	fc := func(string) ([]byte, error) { return []byte("busy"), nil }
	o := New(Options{
		Capture:      fc,
		FastTick:     50 * time.Millisecond,
		SlowTick:     1 * time.Second,
		PerPaneFloor: DisablePerPaneFloor,
	})
	o.Register(Pane{
		SessionID: "s",
		Target:    "s:0.0",
		StatusFn:  func() string { return "working" },
		Sink:      make(chan Signal, 1),
	})
	next := o.poll(time.Now())
	if next > 100*time.Millisecond {
		t.Errorf("expected fast tier, got %v", next)
	}
}

func TestRegisterCancelRemovesPane(t *testing.T) {
	fc := func(string) ([]byte, error) { return []byte("x"), nil }
	o := New(Options{Capture: fc})
	cancel := o.Register(Pane{SessionID: "s", Target: "s:0.0", Sink: make(chan Signal, 1)})
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
	o := New(Options{Capture: fc, PerPaneFloor: DisablePerPaneFloor})
	o.Register(Pane{SessionID: "s", Target: "s:0.0", Sink: make(chan Signal, 1)})

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
	fc := &fakeCapture{plans: map[string][][]byte{
		"s:0.0": {[]byte("stable\n")},
	}}
	// Unbuffered sink with no reader: every stable emit is forced to drop.
	sink := make(chan Signal)
	o := New(Options{
		Capture:         fc.capture,
		StabilityWindow: 100 * time.Millisecond,
		PerPaneFloor:    DisablePerPaneFloor,
	})
	o.Register(Pane{SessionID: "s", Target: "s:0.0", Sink: sink})

	base := time.Now()
	// First poll seeds the hash; next polls find it unchanged and, once the
	// stability window elapses, emit one signal — which must fail to send.
	for i := range 3 {
		o.poll(base.Add(time.Duration(i) * 80 * time.Millisecond))
	}

	if got := o.Metrics().SinkDrops; got == 0 {
		t.Errorf("SinkDrops = 0, want >= 1 for unread sink")
	}
}

func TestStripAnimatedGlyphsFastPathNoE2(t *testing.T) {
	in := []byte("plain ascii line with no special bytes\n")
	out := stripAnimatedGlyphs(in)
	// Fast path must return the input unchanged and share the same backing
	// array (no allocation) when the 0xE2 prefix byte is absent.
	if &in[0] != &out[0] {
		t.Errorf("fast path should return the same slice when input has no 0xE2")
	}
}

func TestStripAnimatedGlyphsHashesEqualAcrossSpinnerFrames(t *testing.T) {
	frame1 := []byte("loading ⠋ tokens\n")
	frame2 := []byte("loading ⠏ tokens\n")
	if string(stripAnimatedGlyphs(frame1)) != string(stripAnimatedGlyphs(frame2)) {
		t.Errorf("spinner frames should hash-equal after strip:\n  %q\n  %q",
			stripAnimatedGlyphs(frame1), stripAnimatedGlyphs(frame2))
	}
}

func TestStripAnimatedGlyphsPreservesBoxBorders(t *testing.T) {
	in := []byte("╭─ Permission ─╮\n│ ❯ Yes       │\n╰──────────────╯\n")
	out := stripAnimatedGlyphs(in)
	if string(out) != string(in) {
		t.Errorf("box-drawing chars must survive strip: got %q", out)
	}
}

func TestPollStaysStableAcrossSpinnerChanges(t *testing.T) {
	fc := &fakeCapture{plans: map[string][][]byte{
		"s:0.0": {
			[]byte("busy ⠋\n"),
			[]byte("busy ⠏\n"),
			[]byte("busy ⠋\n"),
			[]byte("busy ⠏\n"),
		},
	}}
	sink := make(chan Signal, 4)
	o := New(Options{
		Capture:         fc.capture,
		StabilityWindow: 100 * time.Millisecond,
		PerPaneFloor:    DisablePerPaneFloor,
	})
	o.Register(Pane{
		SessionID: "sess-1",
		Target:    "s:0.0",
		StatusFn:  func() string { return "working" },
		Sink:      sink,
	})

	base := time.Now()
	for i := range 4 {
		o.poll(base.Add(time.Duration(i) * 80 * time.Millisecond))
	}

	stables := 0
	for {
		select {
		case sig := <-sink:
			if sig.Stable {
				stables++
			}
		default:
			if stables != 1 {
				t.Errorf("want exactly 1 stable signal across spinner frames, got %d", stables)
			}
			return
		}
	}
}

func TestLastNLines(t *testing.T) {
	tests := []struct {
		in   string
		n    int
		want string
	}{
		{"", 3, ""},
		{"a\nb\nc\n", 2, "b\nc\n"},
		{"a\nb\nc", 2, "b\nc"},
		{"just-one-line", 5, "just-one-line"},
		{"a\nb", 0, "a\nb"},
	}
	for _, tc := range tests {
		got := string(lastNLines([]byte(tc.in), tc.n))
		if got != tc.want {
			t.Errorf("lastNLines(%q, %d) = %q, want %q", tc.in, tc.n, got, tc.want)
		}
	}
}
