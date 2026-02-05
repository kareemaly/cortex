package notifications

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/kareemaly/cortex/internal/daemon/api"
	"github.com/kareemaly/cortex/internal/daemon/config"
	"github.com/kareemaly/cortex/internal/events"
	"github.com/kareemaly/cortex/internal/ticket"
)

// mockChannel captures notifications for testing.
type mockChannel struct {
	mu            sync.Mutex
	notifications []Notification
	available     bool
}

func newMockChannel() *mockChannel {
	return &mockChannel{available: true}
}

func (m *mockChannel) Name() string { return "mock" }

func (m *mockChannel) Available() bool { return m.available }

func (m *mockChannel) Send(ctx context.Context, n Notification) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.notifications = append(m.notifications, n)
	return nil
}

func (m *mockChannel) getNotifications() []Notification {
	m.mu.Lock()
	defer m.mu.Unlock()
	result := make([]Notification, len(m.notifications))
	copy(result, m.notifications)
	return result
}

func (m *mockChannel) clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.notifications = nil
}

func testDispatcherLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: slog.LevelError}))
}

func defaultTestConfig() config.NotificationsConfig {
	return config.NotificationsConfig{
		Channels: config.ChannelsConfig{
			Local: config.LocalChannelConfig{
				Enabled: true,
				Sound:   true,
			},
		},
		Behavior: config.BehaviorConfig{
			BatchWindowSeconds:   1, // Short for testing
			NotifyOnFirstOnly:    false,
			SuppressWhenAttached: false,
		},
		Events: config.EventsConfig{
			AgentWaitingPermission: true,
			AgentIdle:              true,
			AgentError:             true,
			TicketReviewRequested:  true,
		},
	}
}

func setupTestStore(t *testing.T) (string, *ticket.Store, *events.Bus) {
	t.Helper()

	// Create temp project directory
	projectDir := t.TempDir()
	ticketsDir := filepath.Join(projectDir, ".cortex", "tickets")

	bus := events.NewBus()
	store, err := ticket.NewStore(ticketsDir, bus, projectDir)
	if err != nil {
		t.Fatalf("failed to create store: %v", err)
	}

	return projectDir, store, bus
}

func TestDispatcher_NewDispatcher(t *testing.T) {
	cfg := defaultTestConfig()
	ch := newMockChannel()
	bus := events.NewBus()
	logger := testDispatcherLogger()

	d := NewDispatcher(DispatcherConfig{
		Config:   cfg,
		Channels: []Channel{ch},
		Bus:      bus,
		Logger:   logger,
	})

	if d == nil {
		t.Fatal("NewDispatcher returned nil")
	}

	d.Shutdown()
}

func TestDispatcher_Subscribe_Unsubscribe(t *testing.T) {
	cfg := defaultTestConfig()
	ch := newMockChannel()
	bus := events.NewBus()
	logger := testDispatcherLogger()

	d := NewDispatcher(DispatcherConfig{
		Config:   cfg,
		Channels: []Channel{ch},
		Bus:      bus,
		Logger:   logger,
	})
	defer d.Shutdown()

	projectPath := "/test/project"

	// Subscribe
	d.Subscribe(projectPath)
	if len(d.subscriptions) != 1 {
		t.Errorf("expected 1 subscription, got %d", len(d.subscriptions))
	}

	// Double subscribe should be no-op
	d.Subscribe(projectPath)
	if len(d.subscriptions) != 1 {
		t.Errorf("expected 1 subscription after double subscribe, got %d", len(d.subscriptions))
	}

	// Unsubscribe
	d.Unsubscribe(projectPath)
	if len(d.subscriptions) != 0 {
		t.Errorf("expected 0 subscriptions after unsubscribe, got %d", len(d.subscriptions))
	}
}

func TestDispatcher_EventClassification_SessionStatus(t *testing.T) {
	projectDir, store, bus := setupTestStore(t)
	cfg := defaultTestConfig()
	ch := newMockChannel()
	logger := testDispatcherLogger()

	storeManager := api.NewStoreManager(logger, bus)

	d := NewDispatcher(DispatcherConfig{
		Config:       cfg,
		Channels:     []Channel{ch},
		StoreManager: storeManager,
		Bus:          bus,
		Logger:       logger,
	})
	defer d.Shutdown()

	// Create a ticket and start a session
	tkt, err := store.Create("Test Ticket", "body", "", nil)
	if err != nil {
		t.Fatalf("failed to create ticket: %v", err)
	}

	_, err = store.SetSession(tkt.ID, "claude", "ticket-window", nil, nil)
	if err != nil {
		t.Fatalf("failed to set session: %v", err)
	}

	tests := []struct {
		name       string
		status     ticket.AgentStatus
		wantType   NotifiableEventType
		wantNotify bool
	}{
		{"waiting_permission", ticket.AgentStatusWaitingPermission, EventAgentWaitingPermission, true},
		{"idle", ticket.AgentStatusIdle, EventAgentIdle, true},
		{"error", ticket.AgentStatusError, EventAgentError, true},
		{"in_progress", ticket.AgentStatusInProgress, "", false},
		{"starting", ticket.AgentStatusStarting, "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Update session status
			err := store.UpdateSessionStatus(tkt.ID, tt.status, nil, nil)
			if err != nil {
				t.Fatalf("failed to update status: %v", err)
			}

			event := events.Event{
				Type:        events.SessionStatus,
				ProjectPath: projectDir,
				TicketID:    tkt.ID,
			}

			notifiable := d.classifyEvent(event)

			if tt.wantNotify {
				if notifiable == nil {
					t.Errorf("expected notifiable event, got nil")
					return
				}
				if notifiable.Type != tt.wantType {
					t.Errorf("expected type %s, got %s", tt.wantType, notifiable.Type)
				}
			} else {
				if notifiable != nil {
					t.Errorf("expected nil, got event type %s", notifiable.Type)
				}
			}
		})
	}
}

func TestDispatcher_EventClassification_CommentAdded(t *testing.T) {
	projectDir, store, bus := setupTestStore(t)
	cfg := defaultTestConfig()
	ch := newMockChannel()
	logger := testDispatcherLogger()

	storeManager := api.NewStoreManager(logger, bus)

	d := NewDispatcher(DispatcherConfig{
		Config:       cfg,
		Channels:     []Channel{ch},
		StoreManager: storeManager,
		Bus:          bus,
		Logger:       logger,
	})
	defer d.Shutdown()

	// Create a ticket
	tkt, err := store.Create("Test Ticket", "body", "", nil)
	if err != nil {
		t.Fatalf("failed to create ticket: %v", err)
	}

	// Add a review_requested comment
	_, err = store.AddComment(tkt.ID, "", ticket.CommentReviewRequested, "Please review", nil)
	if err != nil {
		t.Fatalf("failed to add comment: %v", err)
	}

	event := events.Event{
		Type:        events.CommentAdded,
		ProjectPath: projectDir,
		TicketID:    tkt.ID,
	}

	notifiable := d.classifyEvent(event)
	if notifiable == nil {
		t.Fatal("expected notifiable event, got nil")
		return
	}
	if notifiable.Type != EventReviewRequested {
		t.Errorf("expected type %s, got %s", EventReviewRequested, notifiable.Type)
	}
}

func TestDispatcher_EventClassification_CommentAdded_NonReview(t *testing.T) {
	projectDir, store, bus := setupTestStore(t)
	cfg := defaultTestConfig()
	ch := newMockChannel()
	logger := testDispatcherLogger()

	storeManager := api.NewStoreManager(logger, bus)

	d := NewDispatcher(DispatcherConfig{
		Config:       cfg,
		Channels:     []Channel{ch},
		StoreManager: storeManager,
		Bus:          bus,
		Logger:       logger,
	})
	defer d.Shutdown()

	// Create a ticket
	tkt, err := store.Create("Test Ticket", "body", "", nil)
	if err != nil {
		t.Fatalf("failed to create ticket: %v", err)
	}

	// Add a general comment (not review_requested)
	_, err = store.AddComment(tkt.ID, "", ticket.CommentGeneral, "Just a comment", nil)
	if err != nil {
		t.Fatalf("failed to add comment: %v", err)
	}

	event := events.Event{
		Type:        events.CommentAdded,
		ProjectPath: projectDir,
		TicketID:    tkt.ID,
	}

	notifiable := d.classifyEvent(event)
	if notifiable != nil {
		t.Errorf("expected nil for non-review comment, got type %s", notifiable.Type)
	}
}

func TestDispatcher_ConfigFiltering(t *testing.T) {
	cfg := defaultTestConfig()
	cfg.Events.AgentIdle = false // Disable idle notifications

	d := NewDispatcher(DispatcherConfig{
		Config:   cfg,
		Channels: []Channel{newMockChannel()},
		Bus:      events.NewBus(),
		Logger:   testDispatcherLogger(),
	})
	defer d.Shutdown()

	tests := []struct {
		eventType NotifiableEventType
		enabled   bool
	}{
		{EventAgentWaitingPermission, true},
		{EventAgentIdle, false}, // Disabled
		{EventAgentError, true},
		{EventReviewRequested, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.eventType), func(t *testing.T) {
			if d.isEventEnabled(tt.eventType) != tt.enabled {
				t.Errorf("isEventEnabled(%s) = %v, want %v", tt.eventType, !tt.enabled, tt.enabled)
			}
		})
	}
}

func TestDispatcher_AttentionTracking(t *testing.T) {
	cfg := defaultTestConfig()
	d := NewDispatcher(DispatcherConfig{
		Config:   cfg,
		Channels: []Channel{newMockChannel()},
		Bus:      events.NewBus(),
		Logger:   testDispatcherLogger(),
	})
	defer d.Shutdown()

	project := "/test/project"

	// First attention should return wasZero=true
	wasZero := d.updateAttention(project, "ticket-1")
	if !wasZero {
		t.Error("first attention should return wasZero=true")
	}

	// Second attention should return wasZero=false
	wasZero = d.updateAttention(project, "ticket-2")
	if wasZero {
		t.Error("second attention should return wasZero=false")
	}

	// Same ticket should return wasZero=false
	wasZero = d.updateAttention(project, "ticket-1")
	if wasZero {
		t.Error("same ticket attention should return wasZero=false")
	}

	// Clear one ticket
	d.clearAttention(project, "ticket-1")

	// Total should be 1
	if count := d.totalAttentionCount(); count != 1 {
		t.Errorf("totalAttentionCount() = %d, want 1", count)
	}

	// Clear last ticket
	d.clearAttention(project, "ticket-2")

	// Total should be 0
	if count := d.totalAttentionCount(); count != 0 {
		t.Errorf("totalAttentionCount() = %d, want 0", count)
	}

	// Next attention should return wasZero=true again
	wasZero = d.updateAttention(project, "ticket-3")
	if !wasZero {
		t.Error("attention after clearing all should return wasZero=true")
	}
}

func TestDispatcher_NotifyOnFirstOnly(t *testing.T) {
	projectDir, store, bus := setupTestStore(t)

	cfg := defaultTestConfig()
	cfg.Behavior.NotifyOnFirstOnly = true
	cfg.Behavior.BatchWindowSeconds = 0 // Immediate flush

	ch := newMockChannel()
	logger := testDispatcherLogger()

	storeManager := api.NewStoreManager(logger, bus)

	d := NewDispatcher(DispatcherConfig{
		Config:       cfg,
		Channels:     []Channel{ch},
		StoreManager: storeManager,
		Bus:          bus,
		Logger:       logger,
	})
	defer d.Shutdown()

	d.Subscribe(projectDir)

	// Create first ticket and trigger notification
	tkt1, _ := store.Create("Ticket 1", "body", "", nil)
	_, _ = store.SetSession(tkt1.ID, "claude", "window1", nil, nil)

	// Clear any previous notifications
	ch.clear()

	// First notification should go through
	_ = store.UpdateSessionStatus(tkt1.ID, ticket.AgentStatusIdle, nil, nil)

	// Wait for processing
	time.Sleep(100 * time.Millisecond)

	notifications := ch.getNotifications()
	if len(notifications) != 1 {
		t.Errorf("expected 1 notification for first attention, got %d", len(notifications))
	}

	ch.clear()

	// Create second ticket - should NOT trigger notification (not first)
	tkt2, _ := store.Create("Ticket 2", "body", "", nil)
	_, _ = store.SetSession(tkt2.ID, "claude", "window2", nil, nil)
	_ = store.UpdateSessionStatus(tkt2.ID, ticket.AgentStatusIdle, nil, nil)

	time.Sleep(100 * time.Millisecond)

	notifications = ch.getNotifications()
	if len(notifications) != 0 {
		t.Errorf("expected 0 notifications for subsequent attention, got %d", len(notifications))
	}
}

func TestDispatcher_BatchWindow(t *testing.T) {
	projectDir, store, bus := setupTestStore(t)

	cfg := defaultTestConfig()
	cfg.Behavior.BatchWindowSeconds = 1 // 1 second batch window
	cfg.Behavior.NotifyOnFirstOnly = false

	ch := newMockChannel()
	logger := testDispatcherLogger()

	storeManager := api.NewStoreManager(logger, bus)

	d := NewDispatcher(DispatcherConfig{
		Config:       cfg,
		Channels:     []Channel{ch},
		StoreManager: storeManager,
		Bus:          bus,
		Logger:       logger,
	})
	defer d.Shutdown()

	d.Subscribe(projectDir)

	// Create multiple tickets quickly
	for i := 0; i < 3; i++ {
		tkt, _ := store.Create("Ticket", "body", "", nil)
		_, _ = store.SetSession(tkt.ID, "claude", "window", nil, nil)
		_ = store.UpdateSessionStatus(tkt.ID, ticket.AgentStatusIdle, nil, nil)
	}

	// Should not have notifications yet (within batch window)
	time.Sleep(100 * time.Millisecond)
	if len(ch.getNotifications()) > 0 {
		t.Error("expected no notifications within batch window")
	}

	// Wait for batch window to expire
	time.Sleep(1100 * time.Millisecond)

	notifications := ch.getNotifications()
	if len(notifications) != 1 {
		t.Errorf("expected 1 batched notification, got %d", len(notifications))
	}
}

func TestDispatcher_FormatNotification_Single(t *testing.T) {
	cfg := defaultTestConfig()
	d := NewDispatcher(DispatcherConfig{
		Config:   cfg,
		Channels: []Channel{newMockChannel()},
		Bus:      events.NewBus(),
		Logger:   testDispatcherLogger(),
	})
	defer d.Shutdown()

	event := NotifiableEvent{
		Type:        EventAgentWaitingPermission,
		ProjectPath: "/test",
		TicketID:    "123",
		TicketTitle: "Fix the bug",
	}

	notification := d.formatNotification([]NotifiableEvent{event})

	if notification.Title != "Agent Waiting for Permission" {
		t.Errorf("Title = %q, want 'Agent Waiting for Permission'", notification.Title)
	}
	if notification.Body != "Fix the bug" {
		t.Errorf("Body = %q, want 'Fix the bug'", notification.Body)
	}
	if !notification.Sound {
		t.Error("Sound should be true based on config")
	}
}

func TestDispatcher_FormatNotification_Batch(t *testing.T) {
	cfg := defaultTestConfig()
	d := NewDispatcher(DispatcherConfig{
		Config:   cfg,
		Channels: []Channel{newMockChannel()},
		Bus:      events.NewBus(),
		Logger:   testDispatcherLogger(),
	})
	defer d.Shutdown()

	events := []NotifiableEvent{
		{Type: EventAgentIdle, TicketTitle: "Ticket A"},
		{Type: EventAgentIdle, TicketTitle: "Ticket B"},
		{Type: EventAgentError, TicketTitle: "Ticket C"},
	}

	notification := d.formatNotification(events)

	if notification.Title != "3 agents need attention" {
		t.Errorf("Title = %q, want '3 agents need attention'", notification.Title)
	}
}

func TestDispatcher_FormatNotification_BatchWithMore(t *testing.T) {
	cfg := defaultTestConfig()
	d := NewDispatcher(DispatcherConfig{
		Config:   cfg,
		Channels: []Channel{newMockChannel()},
		Bus:      events.NewBus(),
		Logger:   testDispatcherLogger(),
	})
	defer d.Shutdown()

	events := []NotifiableEvent{
		{Type: EventAgentIdle, TicketTitle: "Ticket A"},
		{Type: EventAgentIdle, TicketTitle: "Ticket B"},
		{Type: EventAgentIdle, TicketTitle: "Ticket C"},
		{Type: EventAgentIdle, TicketTitle: "Ticket D"},
	}

	notification := d.formatNotification(events)

	if notification.Title != "4 agents need attention" {
		t.Errorf("Title = %q, want '4 agents need attention'", notification.Title)
	}
	// Body should mention "and 2 more"
	if notification.Body != "Ticket A and Ticket B, and 2 more" {
		t.Errorf("Body = %q, want 'Ticket A and Ticket B, and 2 more'", notification.Body)
	}
}

func TestDispatcher_Urgency(t *testing.T) {
	cfg := defaultTestConfig()
	d := NewDispatcher(DispatcherConfig{
		Config:   cfg,
		Channels: []Channel{newMockChannel()},
		Bus:      events.NewBus(),
		Logger:   testDispatcherLogger(),
	})
	defer d.Shutdown()

	tests := []struct {
		eventType   NotifiableEventType
		wantUrgency string
	}{
		{EventAgentError, "critical"},
		{EventAgentWaitingPermission, "normal"},
		{EventAgentIdle, "low"},
		{EventReviewRequested, "low"},
	}

	for _, tt := range tests {
		t.Run(string(tt.eventType), func(t *testing.T) {
			urgency := d.urgencyForEvent(tt.eventType)
			if urgency != tt.wantUrgency {
				t.Errorf("urgencyForEvent(%s) = %q, want %q", tt.eventType, urgency, tt.wantUrgency)
			}
		})
	}
}

func TestJoinTitles(t *testing.T) {
	tests := []struct {
		name   string
		titles []string
		want   string
	}{
		{"empty", []string{}, ""},
		{"one", []string{"A"}, "A"},
		{"two", []string{"A", "B"}, "A and B"},
		{"three", []string{"A", "B", "C"}, "A, B, and C"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := joinTitles(tt.titles)
			if got != tt.want {
				t.Errorf("joinTitles(%v) = %q, want %q", tt.titles, got, tt.want)
			}
		})
	}
}

func TestDispatcher_ImplementsGracefulShutdown(t *testing.T) {
	cfg := defaultTestConfig()
	ch := newMockChannel()
	bus := events.NewBus()
	logger := testDispatcherLogger()

	d := NewDispatcher(DispatcherConfig{
		Config:   cfg,
		Channels: []Channel{ch},
		Bus:      bus,
		Logger:   logger,
	})

	d.Subscribe("/project1")
	d.Subscribe("/project2")

	// Shutdown should complete without hanging
	done := make(chan struct{})
	go func() {
		d.Shutdown()
		close(done)
	}()

	select {
	case <-done:
		// Success
	case <-time.After(5 * time.Second):
		t.Fatal("Shutdown did not complete in time")
	}
}
