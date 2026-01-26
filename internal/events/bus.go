package events

import "sync"

// EventType represents the type of event emitted by the system.
type EventType string

const (
	TicketCreated   EventType = "ticket_created"
	TicketUpdated   EventType = "ticket_updated"
	TicketDeleted   EventType = "ticket_deleted"
	TicketMoved     EventType = "ticket_moved"
	SessionStarted  EventType = "session_started"
	SessionEnded    EventType = "session_ended"
	SessionStatus   EventType = "session_status"
	CommentAdded    EventType = "comment_added"
	ReviewRequested EventType = "review_requested"
)

// Event represents a change in the system.
type Event struct {
	Type        EventType `json:"type"`
	ProjectPath string    `json:"project_path"`
	TicketID    string    `json:"ticket_id"`
	Payload     any       `json:"payload,omitempty"`
}

// Bus is an in-process pub/sub event bus keyed by project path.
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

// Emit sends an event to all subscribers for the event's project path.
// Non-blocking: slow consumers will miss events.
func (b *Bus) Emit(e Event) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	subs, ok := b.subscribers[e.ProjectPath]
	if !ok {
		return
	}

	for sub := range subs {
		select {
		case sub.ch <- e:
		default:
		}
	}
}

// Subscribe registers a listener for events on the given project path.
// Returns a receive-only channel and an unsubscribe function.
func (b *Bus) Subscribe(projectPath string) (<-chan Event, func()) {
	sub := &subscriber{
		ch: make(chan Event, 64),
	}

	b.mu.Lock()
	subs, ok := b.subscribers[projectPath]
	if !ok {
		subs = make(map[*subscriber]struct{})
		b.subscribers[projectPath] = subs
	}
	subs[sub] = struct{}{}
	b.mu.Unlock()

	unsubscribe := func() {
		b.mu.Lock()
		defer b.mu.Unlock()

		if subs, ok := b.subscribers[projectPath]; ok {
			delete(subs, sub)
			if len(subs) == 0 {
				delete(b.subscribers, projectPath)
			}
		}
		close(sub.ch)
	}

	return sub.ch, unsubscribe
}
