package api

import (
	"context"
	"log/slog"
	"net/http"
	"sync"

	"github.com/hiveryn/agentruntime"
	"github.com/hiveryn/agentruntime/adapter/claude"
	"github.com/hiveryn/agentruntime/adapter/codex"
	"github.com/hiveryn/agentruntime/adapter/opencode"
	"github.com/hiveryn/agentruntime/ingest"
	"github.com/kareemaly/cortex/internal/core/agent"
	"github.com/kareemaly/cortex/internal/session"
)

type receiverEntry struct {
	receiver *ingest.Receiver
	adapter  agentruntime.Adapter
}

// ReceiverManager wraps per-agent ingest receivers and provides
// normalized event access by Cortex session UUID (Event.ID).
type ReceiverManager struct {
	receivers map[agentruntime.AgentKind]*receiverEntry
	cache     sync.Map // map[string]agentruntime.Event, key = Event.ID (Cortex session UUID)
	logger    *slog.Logger
}

// NewReceiverManager creates receivers for claude, codex, and opencode.
func NewReceiverManager(logger *slog.Logger) *ReceiverManager {
	rm := &ReceiverManager{
		receivers: map[agentruntime.AgentKind]*receiverEntry{},
		logger:    logger,
	}

	for _, entry := range []struct {
		adapter agentruntime.Adapter
	}{
		{claude.New(claude.DefaultOptions())},
		{codex.New(codex.DefaultOptions())},
		{opencode.New(opencode.DefaultOptions())},
	} {
		rec := ingest.NewReceiver(entry.adapter)
		rm.receivers[entry.adapter.Agent()] = &receiverEntry{
			receiver: rec,
			adapter:  entry.adapter,
		}
	}

	return rm
}

// StartEventLoop subscribes to all receiver hubs and populates the cache
// by Event.ID. Runs until ctx is cancelled.
func (m *ReceiverManager) StartEventLoop(ctx context.Context) {
	if m == nil {
		return
	}
	for _, entry := range m.receivers {
		sub := entry.receiver.Hub().Subscribe(ingest.Filter{})
		go func(sub *ingest.Subscription) {
			defer sub.Close()
			for {
				select {
				case ev, ok := <-sub.Events:
					if !ok {
						return
					}
					if ev.ID != "" {
						m.cache.Store(ev.ID, ev)
					}
				case <-ctx.Done():
					return
				}
			}
		}(sub)
	}
}

// GetEvent returns the latest cached normalized event for the given
// Cortex session UUID (Event.ID).
func (m *ReceiverManager) GetEvent(cortexSessionID string) (agentruntime.Event, bool) {
	if m == nil || cortexSessionID == "" {
		return agentruntime.Event{}, false
	}
	v, ok := m.cache.Load(cortexSessionID)
	if !ok {
		return agentruntime.Event{}, false
	}
	return v.(agentruntime.Event), true
}

// Ingest routes a raw hook payload to the appropriate receiver.
func (m *ReceiverManager) Ingest(ctx context.Context, agentKind agentruntime.AgentKind, data []byte) (*agentruntime.Event, error) {
	if m == nil {
		return nil, nil
	}
	entry, ok := m.receivers[agentKind]
	if !ok {
		return nil, nil
	}
	return entry.receiver.Ingest(ctx, agentKind, data)
}

// Handler returns an HTTP handler for hook ingestion for the given agent.
func (m *ReceiverManager) Handler(agentKind agentruntime.AgentKind) http.Handler {
	if m == nil {
		return nil
	}
	entry, ok := m.receivers[agentKind]
	if !ok {
		return nil
	}
	return entry.receiver.Handler(agentKind)
}

// EventsFor returns a channel of HubEvent filtered to the given Cortex session UUID.
// The channel is closed when ctx is cancelled or the subscription is closed.
func (m *ReceiverManager) EventsFor(ctx context.Context, cortexSessionID string) <-chan agent.HubEvent {
	if m == nil || cortexSessionID == "" {
		return nil
	}
	return m.eventsFromAllReceivers(ctx, cortexSessionID)
}

// eventsFromAllReceivers subscribes to all receivers' hubs with a filter
// on Event.ID and merges the streams. In practice only one receiver will
// produce events for a given session UUID.
func (m *ReceiverManager) eventsFromAllReceivers(ctx context.Context, cortexSessionID string) <-chan agent.HubEvent {
	ch := make(chan agent.HubEvent, 32)
	var wg sync.WaitGroup

	for _, entry := range m.receivers {
		sub := entry.receiver.Hub().Subscribe(ingest.Filter{ID: cortexSessionID})
		wg.Add(1)
		go func(sub *ingest.Subscription) {
			defer wg.Done()
			defer sub.Close()
			for {
				select {
				case ev, ok := <-sub.Events:
					if !ok {
						return
					}
					out := agent.HubEvent{
						Status: session.AgentStatus(string(ev.Status)),
						Tool:   ev.Tool,
						Work:   ev.Message,
					}
					select {
					case ch <- out:
					case <-ctx.Done():
						return
					}
				case <-ctx.Done():
					return
				}
			}
		}(sub)
	}

	go func() {
		wg.Wait()
		close(ch)
	}()

	return ch
}

// Close stops all event loop subscriptions. The cache is preserved for
// final polling from API handlers.
func (m *ReceiverManager) Close() {
	// Subscriptions are closed by their owning goroutines via context cancellation.
	// The StartEventLoop caller should cancel the context to trigger teardown.
}

// hubEventSource returns the ReceiverManager's EventsFor method as a function
// suitable for spawn.Dependencies.HubEventSource.
func hubEventSource(rm *ReceiverManager) func(ctx context.Context, agentSessionID string) <-chan agent.HubEvent {
	if rm == nil {
		return nil
	}
	return rm.EventsFor
}
