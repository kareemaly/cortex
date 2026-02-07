package notifications

import (
	"context"
	"fmt"
	"log/slog"
	"path/filepath"
	"sync"
	"time"

	"github.com/kareemaly/cortex/internal/daemon/api"
	"github.com/kareemaly/cortex/internal/daemon/config"
	"github.com/kareemaly/cortex/internal/events"
	"github.com/kareemaly/cortex/internal/session"
	"github.com/kareemaly/cortex/internal/storage"
	"github.com/kareemaly/cortex/internal/ticket"
	"github.com/kareemaly/cortex/internal/tmux"
)

// NotifiableEventType represents the type of notifiable event.
type NotifiableEventType string

const (
	EventAgentWaitingPermission NotifiableEventType = "agent_waiting_permission"
	EventAgentIdle              NotifiableEventType = "agent_idle"
	EventAgentError             NotifiableEventType = "agent_error"
	EventReviewRequested        NotifiableEventType = "review_requested"
)

// NotifiableEvent represents an event that may trigger a notification.
type NotifiableEvent struct {
	Type        NotifiableEventType
	ProjectPath string
	TicketID    string
	TicketTitle string
	TmuxWindow  string
}

// DispatcherConfig holds the configuration for creating a Dispatcher.
type DispatcherConfig struct {
	Config         config.NotificationsConfig
	Channels       []Channel
	StoreManager   *api.StoreManager
	SessionManager *api.SessionManager
	TmuxManager    *tmux.Manager
	Bus            *events.Bus
	Logger         *slog.Logger
}

// Dispatcher subscribes to the event bus and routes notifications to channels.
type Dispatcher struct {
	config         config.NotificationsConfig
	channels       []Channel
	storeManager   *api.StoreManager
	sessionManager *api.SessionManager
	tmuxManager    *tmux.Manager
	bus            *events.Bus
	logger         *slog.Logger

	// Subscription management
	subsMu        sync.Mutex
	subscriptions map[string]func() // projectPath -> unsubscribe

	// Attention tracking (global across all projects)
	attnMu           sync.Mutex
	attentionTickets map[string]map[string]struct{} // project -> ticketIDs

	// Batching
	batchMu       sync.Mutex
	pendingEvents []NotifiableEvent
	batchTimer    *time.Timer

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

// NewDispatcher creates a new notification dispatcher.
func NewDispatcher(cfg DispatcherConfig) *Dispatcher {
	ctx, cancel := context.WithCancel(context.Background())

	d := &Dispatcher{
		config:           cfg.Config,
		channels:         cfg.Channels,
		storeManager:     cfg.StoreManager,
		sessionManager:   cfg.SessionManager,
		tmuxManager:      cfg.TmuxManager,
		bus:              cfg.Bus,
		logger:           cfg.Logger,
		subscriptions:    make(map[string]func()),
		attentionTickets: make(map[string]map[string]struct{}),
		ctx:              ctx,
		cancel:           cancel,
	}

	return d
}

// Subscribe starts listening to events for a project.
func (d *Dispatcher) Subscribe(projectPath string) {
	d.subsMu.Lock()
	defer d.subsMu.Unlock()

	// Already subscribed
	if _, exists := d.subscriptions[projectPath]; exists {
		return
	}

	eventCh, unsubscribe := d.bus.Subscribe(projectPath)
	d.subscriptions[projectPath] = unsubscribe

	d.wg.Add(1)
	go d.processEvents(projectPath, eventCh)

	d.logger.Debug("subscribed to project events", "project", projectPath)
}

// Unsubscribe stops listening to events for a project.
func (d *Dispatcher) Unsubscribe(projectPath string) {
	d.subsMu.Lock()
	defer d.subsMu.Unlock()

	if unsubscribe, exists := d.subscriptions[projectPath]; exists {
		unsubscribe()
		delete(d.subscriptions, projectPath)
		d.logger.Debug("unsubscribed from project events", "project", projectPath)
	}
}

// Shutdown gracefully stops the dispatcher.
func (d *Dispatcher) Shutdown() {
	d.cancel()

	// Unsubscribe from all projects
	d.subsMu.Lock()
	for path, unsubscribe := range d.subscriptions {
		unsubscribe()
		delete(d.subscriptions, path)
	}
	d.subsMu.Unlock()

	// Stop batch timer
	d.batchMu.Lock()
	if d.batchTimer != nil {
		d.batchTimer.Stop()
	}
	d.batchMu.Unlock()

	d.wg.Wait()
	d.logger.Info("notification dispatcher stopped")
}

// processEvents reads events from a project's channel and handles them.
func (d *Dispatcher) processEvents(projectPath string, eventCh <-chan events.Event) {
	defer d.wg.Done()

	for {
		select {
		case <-d.ctx.Done():
			return
		case event, ok := <-eventCh:
			if !ok {
				return
			}
			d.handleEvent(event)
		}
	}
}

// handleEvent processes a single event from the bus.
func (d *Dispatcher) handleEvent(event events.Event) {
	notifiable := d.classifyEvent(event)
	if notifiable == nil {
		return
	}

	// Check if this event type is enabled
	if !d.isEventEnabled(notifiable.Type) {
		d.logger.Debug("event type disabled", "type", notifiable.Type)
		return
	}

	// Check attachment suppression
	if d.config.Behavior.SuppressWhenAttached && d.isUserAttached(event.ProjectPath, notifiable.TmuxWindow) {
		d.logger.Debug("notification suppressed, user attached", "window", notifiable.TmuxWindow)
		return
	}

	// Update attention tracking
	wasZero := d.updateAttention(event.ProjectPath, event.TicketID)

	// Check notify_on_first_only
	if d.config.Behavior.NotifyOnFirstOnly && !wasZero {
		d.logger.Debug("notification suppressed, not first attention", "ticket", event.TicketID)
		return
	}

	d.addToBatch(*notifiable)
}

// classifyEvent maps a bus event to a notifiable event, or returns nil if not notifiable.
func (d *Dispatcher) classifyEvent(event events.Event) *NotifiableEvent {
	switch event.Type {
	case events.SessionStatus:
		return d.classifySessionStatus(event)
	case events.CommentAdded:
		return d.classifyCommentAdded(event)
	default:
		return nil
	}
}

// classifySessionStatus checks if a session status event is notifiable.
func (d *Dispatcher) classifySessionStatus(event events.Event) *NotifiableEvent {
	// Fetch ticket to get title
	store, err := d.storeManager.GetStore(event.ProjectPath)
	if err != nil {
		d.logger.Debug("failed to get store", "error", err)
		return nil
	}

	t, _, err := store.Get(event.TicketID)
	if err != nil {
		d.logger.Debug("failed to get ticket", "error", err)
		return nil
	}

	// Look up session from session manager
	var sess *session.Session
	if d.sessionManager != nil {
		sessStore := d.sessionManager.GetStore(event.ProjectPath)
		shortID := storage.ShortID(event.TicketID)
		sess, _ = sessStore.Get(shortID)
	}

	if sess == nil {
		return nil
	}

	status := sess.Status

	// Clear attention if agent becomes active
	if status == session.AgentStatusInProgress || status == session.AgentStatusStarting {
		d.clearAttention(event.ProjectPath, event.TicketID)
		return nil
	}

	var eventType NotifiableEventType
	switch status {
	case session.AgentStatusWaitingPermission:
		eventType = EventAgentWaitingPermission
	case session.AgentStatusIdle:
		eventType = EventAgentIdle
	case session.AgentStatusError:
		eventType = EventAgentError
	default:
		return nil
	}

	return &NotifiableEvent{
		Type:        eventType,
		ProjectPath: event.ProjectPath,
		TicketID:    event.TicketID,
		TicketTitle: t.Title,
		TmuxWindow:  sess.TmuxWindow,
	}
}

// classifyCommentAdded checks if a comment event indicates a review request.
func (d *Dispatcher) classifyCommentAdded(event events.Event) *NotifiableEvent {
	store, err := d.storeManager.GetStore(event.ProjectPath)
	if err != nil {
		return nil
	}

	t, _, err := store.Get(event.TicketID)
	if err != nil {
		return nil
	}

	// Check if latest comment is a review request
	if len(t.Comments) == 0 {
		return nil
	}

	latest := t.Comments[len(t.Comments)-1]
	if latest.Type != ticket.CommentReviewRequested {
		return nil
	}

	// Look up session for tmux window
	var tmuxWindow string
	if d.sessionManager != nil {
		sessStore := d.sessionManager.GetStore(event.ProjectPath)
		shortID := storage.ShortID(event.TicketID)
		sess, _ := sessStore.Get(shortID)
		if sess != nil {
			tmuxWindow = sess.TmuxWindow
		}
	}

	return &NotifiableEvent{
		Type:        EventReviewRequested,
		ProjectPath: event.ProjectPath,
		TicketID:    event.TicketID,
		TicketTitle: t.Title,
		TmuxWindow:  tmuxWindow,
	}
}

// isEventEnabled returns true if the event type is enabled in config.
func (d *Dispatcher) isEventEnabled(eventType NotifiableEventType) bool {
	switch eventType {
	case EventAgentWaitingPermission:
		return d.config.Events.AgentWaitingPermission
	case EventAgentIdle:
		return d.config.Events.AgentIdle
	case EventAgentError:
		return d.config.Events.AgentError
	case EventReviewRequested:
		return d.config.Events.TicketReviewRequested
	default:
		return false
	}
}

// isUserAttached checks if a user is viewing the specified tmux window.
func (d *Dispatcher) isUserAttached(projectPath, windowName string) bool {
	if d.tmuxManager == nil || windowName == "" {
		return false
	}

	session := filepath.Base(projectPath)
	return d.tmuxManager.IsUserAttached(session, windowName)
}

// updateAttention adds a ticket to the attention set.
// Returns true if this was the first ticket needing attention (was zero before).
func (d *Dispatcher) updateAttention(projectPath, ticketID string) bool {
	d.attnMu.Lock()
	defer d.attnMu.Unlock()

	// Count total before adding
	totalBefore := d.totalAttentionCount()

	// Add to attention set
	if d.attentionTickets[projectPath] == nil {
		d.attentionTickets[projectPath] = make(map[string]struct{})
	}
	d.attentionTickets[projectPath][ticketID] = struct{}{}

	return totalBefore == 0
}

// clearAttention removes a ticket from the attention set.
func (d *Dispatcher) clearAttention(projectPath, ticketID string) {
	d.attnMu.Lock()
	defer d.attnMu.Unlock()

	if tickets, exists := d.attentionTickets[projectPath]; exists {
		delete(tickets, ticketID)
		if len(tickets) == 0 {
			delete(d.attentionTickets, projectPath)
		}
	}
}

// totalAttentionCount returns the total number of tickets needing attention.
func (d *Dispatcher) totalAttentionCount() int {
	count := 0
	for _, tickets := range d.attentionTickets {
		count += len(tickets)
	}
	return count
}

// addToBatch adds an event to the pending batch and starts the timer if needed.
func (d *Dispatcher) addToBatch(event NotifiableEvent) {
	d.batchMu.Lock()
	d.pendingEvents = append(d.pendingEvents, event)

	batchWindow := time.Duration(d.config.Behavior.BatchWindowSeconds) * time.Second

	// If batch window is 0 or negative, flush immediately
	if batchWindow <= 0 {
		d.batchMu.Unlock()
		d.flushBatch()
		return
	}

	// Start timer if not running
	if d.batchTimer == nil {
		d.batchTimer = time.AfterFunc(batchWindow, d.flushBatch)
	}
	d.batchMu.Unlock()
}

// flushBatch sends all pending events as a notification.
func (d *Dispatcher) flushBatch() {
	d.batchMu.Lock()
	eventsToSend := d.pendingEvents
	d.pendingEvents = nil
	d.batchTimer = nil
	d.batchMu.Unlock()

	if len(eventsToSend) == 0 {
		return
	}

	notification := d.formatNotification(eventsToSend)

	for _, ch := range d.channels {
		if err := ch.Send(d.ctx, notification); err != nil {
			d.logger.Error("failed to send notification", "channel", ch.Name(), "error", err)
		}
	}
}

// formatNotification creates a notification from a batch of events.
func (d *Dispatcher) formatNotification(evts []NotifiableEvent) Notification {
	if len(evts) == 1 {
		return d.formatSingleNotification(evts[0])
	}
	return d.formatBatchNotification(evts)
}

// formatSingleNotification creates a notification for a single event.
func (d *Dispatcher) formatSingleNotification(event NotifiableEvent) Notification {
	var title string
	switch event.Type {
	case EventAgentWaitingPermission:
		title = "Agent Waiting for Permission"
	case EventAgentIdle:
		title = "Agent Idle"
	case EventAgentError:
		title = "Agent Error"
	case EventReviewRequested:
		title = "Review Requested"
	default:
		title = "Agent Needs Attention"
	}

	return Notification{
		Title:   title,
		Body:    event.TicketTitle,
		Sound:   d.config.Channels.Local.Sound,
		Urgency: d.urgencyForEvent(event.Type),
	}
}

// formatBatchNotification creates a notification for multiple events.
func (d *Dispatcher) formatBatchNotification(evts []NotifiableEvent) Notification {
	title := fmt.Sprintf("%d agents need attention", len(evts))

	// Build body with ticket titles
	var body string
	if len(evts) <= 3 {
		titles := make([]string, len(evts))
		for i, e := range evts {
			titles[i] = e.TicketTitle
		}
		body = joinTitles(titles)
	} else {
		titles := []string{evts[0].TicketTitle, evts[1].TicketTitle}
		body = fmt.Sprintf("%s, and %d more", joinTitles(titles), len(evts)-2)
	}

	return Notification{
		Title:   title,
		Body:    body,
		Sound:   d.config.Channels.Local.Sound,
		Urgency: "normal",
	}
}

// urgencyForEvent returns the urgency level for an event type.
func (d *Dispatcher) urgencyForEvent(eventType NotifiableEventType) string {
	switch eventType {
	case EventAgentError:
		return "critical"
	case EventAgentWaitingPermission:
		return "normal"
	default:
		return "low"
	}
}

// joinTitles joins titles with commas and "and".
func joinTitles(titles []string) string {
	switch len(titles) {
	case 0:
		return ""
	case 1:
		return titles[0]
	case 2:
		return titles[0] + " and " + titles[1]
	default:
		result := ""
		for i, t := range titles {
			if i == len(titles)-1 {
				result += "and " + t
			} else {
				result += t + ", "
			}
		}
		return result
	}
}
