package events

import (
	"log/slog"
	"sync"
)

// EventType represents the type of event emitted by the system.
type EventType string

const (
	TicketCreated     EventType = "ticket_created"
	TicketUpdated     EventType = "ticket_updated"
	TicketDeleted     EventType = "ticket_deleted"
	TicketMoved       EventType = "ticket_moved"
	SessionStarted    EventType = "session_started"
	SessionEnded      EventType = "session_ended"
	SessionStatus     EventType = "session_status"
	ConclusionCreated EventType = "conclusion_created"
	TicketQueued      EventType = "ticket_queued"
	TicketDequeued    EventType = "ticket_dequeued"
)

// Event represents a change in the system.
type Event struct {
	Type          EventType `json:"type"`
	ArchitectPath string    `json:"architect_path"`
	TicketID      string    `json:"ticket_id"`
	Payload       any       `json:"payload,omitempty"`
}

// Bus is an in-process pub/sub event bus keyed by architect path.
type Bus struct {
	mu          sync.RWMutex
	subscribers map[string]map[*subscriber]struct{}
}

type subscriber struct {
	ch chan Event
}

// NewBus creates a new event bus.
func NewBus() *Bus {
	return &Bus{
		subscribers: make(map[string]map[*subscriber]struct{}),
	}
}

// Emit sends an event to all subscribers for the event's architect path.
// Non-blocking: slow consumers will miss events.
func (b *Bus) Emit(e Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	allSubs := make(map[*subscriber]struct{})

	if subs, ok := b.subscribers[e.ArchitectPath]; ok {
		for sub := range subs {
			allSubs[sub] = struct{}{}
		}
	}

	if e.ArchitectPath != "" {
		if subs, ok := b.subscribers[""]; ok {
			for sub := range subs {
				allSubs[sub] = struct{}{}
			}
		}
	}

	for sub := range allSubs {
		select {
		case sub.ch <- e:
		default:
			slog.Warn("event dropped: subscriber buffer full",
				"type", string(e.Type),
				"architect", e.ArchitectPath,
			)
		}
	}
}

// Subscribe registers a listener for events on the given architect path.
// Returns a receive-only channel and an unsubscribe function.
func (b *Bus) Subscribe(architectPath string) (<-chan Event, func()) {
	sub := &subscriber{
		ch: make(chan Event, 64),
	}

	b.mu.Lock()
	subs, ok := b.subscribers[architectPath]
	if !ok {
		subs = make(map[*subscriber]struct{})
		b.subscribers[architectPath] = subs
	}
	subs[sub] = struct{}{}
	b.mu.Unlock()

	unsubscribe := func() {
		b.mu.Lock()
		defer b.mu.Unlock()

		if subs, ok := b.subscribers[architectPath]; ok {
			delete(subs, sub)
			if len(subs) == 0 {
				delete(b.subscribers, architectPath)
			}
		}
		close(sub.ch)
	}

	return sub.ch, unsubscribe
}
