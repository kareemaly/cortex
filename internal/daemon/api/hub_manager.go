package api

import (
	"context"
	"log/slog"
	"sync"

	"github.com/kareemaly/agentstatus"
	_ "github.com/kareemaly/agentstatus/adapters/claude"
	_ "github.com/kareemaly/agentstatus/adapters/codex"
	_ "github.com/kareemaly/agentstatus/adapters/opencode"
	"github.com/kareemaly/cortex/internal/core/agent"
	"github.com/kareemaly/cortex/internal/session"
)

// HubManager wraps an agentstatus Hub and maintains a sessionCache
// that maps agentstatus SessionID → latest Event. This is the bridge
// between the Hub's event stream and the API response overlay.
type HubManager struct {
	hub          *agentstatus.Hub
	sessionCache sync.Map // map[string]agentstatus.Event, key = agentSessionID
	logger       *slog.Logger
}

// NewHubManager creates a Hub with the given logger and returns a HubManager.
func NewHubManager(logger *slog.Logger) (*HubManager, error) {
	hub, err := agentstatus.NewHub(agentstatus.HubConfig{
		Logger: logger,
	})
	if err != nil {
		return nil, err
	}
	return &HubManager{hub: hub, logger: logger}, nil
}

// StartEventLoop subscribes to hub.Events() and stores the latest Event
// per SessionID into sessionCache. Runs until ctx is cancelled or the hub
// is closed (channel close unblocks the range).
func (m *HubManager) StartEventLoop(ctx context.Context) {
	stream := m.hub.Events()
	ch := stream.Channel()
	go func() {
		for {
			select {
			case ev, ok := <-ch:
				if !ok {
					return
				}
				m.sessionCache.Store(ev.SessionID, ev)
			case <-ctx.Done():
				return
			}
		}
	}()
}

// GetEvent returns the latest cached Event for the given agentstatus SessionID.
func (m *HubManager) GetEvent(agentSessionID string) (agentstatus.Event, bool) {
	v, ok := m.sessionCache.Load(agentSessionID)
	if !ok {
		return agentstatus.Event{}, false
	}
	return v.(agentstatus.Event), true
}

// Ingest forwards a raw hook payload to the Hub.
func (m *HubManager) Ingest(agent agentstatus.Agent, payload []byte) error {
	return m.hub.Ingest(agent, payload)
}

// Close shuts the Hub down and unblocks the event loop goroutine.
func (m *HubManager) Close() error {
	return m.hub.Close()
}

// EventsFor returns a channel of Hub-sourced status events filtered to the
// given agent session ID. The channel is closed when ctx is cancelled or the
// Hub closes. This is used by per-session supervisors to forward Hub status
// transitions to /agent/status → SSE.
func (m *HubManager) EventsFor(ctx context.Context, agentSessionID string) <-chan agent.HubEvent {
	stream := m.hub.Events()
	ch := make(chan agent.HubEvent, 32)
	go func() {
		defer close(ch)
		for {
			select {
			case <-ctx.Done():
				return
			case ev, ok := <-stream.Channel():
				if !ok {
					return
				}
				if ev.SessionID != agentSessionID {
					continue
				}
				out := agent.HubEvent{
					Status: session.AgentStatus(string(ev.Status)),
					Tool:   ev.Tool,
					Work:   ev.Work,
				}
				select {
				case ch <- out:
				case <-ctx.Done():
					return
				}
			}
		}
	}()
	return ch
}

// RegisterOpenCodeSession tags an OpenCode session with the cortex session ID.
// Subsequent Hub events from openCodeSessionID will carry a "cortex_session_id"
// tag, enabling tag-based filtering in future real-time supervisor paths.
func (m *HubManager) RegisterOpenCodeSession(openCodeSessionID, cortexSessionID string) {
	m.hub.RegisterSession(openCodeSessionID, map[string]string{
		"cortex_session_id": cortexSessionID,
	})
}

// hubEventSource returns the HubManager's EventsFor method as a function
// suitable for spawn.Dependencies.HubEventSource. Returns nil when hm is nil
// so supervisors run in liveness-only mode.
func hubEventSource(hm *HubManager) func(ctx context.Context, agentSessionID string) <-chan agent.HubEvent {
	if hm == nil {
		return nil
	}
	return hm.EventsFor
}
