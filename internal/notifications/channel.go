package notifications

import "context"

// Notification represents a desktop notification.
type Notification struct {
	Title   string
	Body    string
	Sound   bool
	Urgency string // "low", "normal", "critical"
}

// Channel defines the interface for notification delivery.
type Channel interface {
	Name() string
	Send(ctx context.Context, n Notification) error
	Available() bool
}
