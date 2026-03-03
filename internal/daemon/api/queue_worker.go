package api

import (
	"context"
	"log/slog"

	"github.com/kareemaly/cortex/internal/architect/config"
	"github.com/kareemaly/cortex/internal/core/spawn"
	"github.com/kareemaly/cortex/internal/events"
	"github.com/kareemaly/cortex/internal/ticket"
)

type QueueWorker struct {
	deps   *Dependencies
	logger *slog.Logger
	bus    *events.Bus
	ctx    context.Context
	cancel context.CancelFunc
}

func NewQueueWorker(deps *Dependencies) *QueueWorker {
	return &QueueWorker{
		deps:   deps,
		logger: deps.Logger,
		bus:    deps.Bus,
	}
}

func (w *QueueWorker) Start(ctx context.Context) {
	w.ctx, w.cancel = context.WithCancel(ctx)

	ch, unsubscribe := w.bus.Subscribe("")
	go func() {
		defer unsubscribe()
		for {
			select {
			case <-w.ctx.Done():
				return
			case event := <-ch:
				if event.Type == events.SessionEnded {
					go w.handleSessionEnded(event.ArchitectPath, event.TicketID)
				}
			}
		}
	}()

	w.logger.Debug("queue worker started")
}

func (w *QueueWorker) Stop() {
	if w.cancel != nil {
		w.cancel()
	}
}

func (w *QueueWorker) handleSessionEnded(projectPath, endedTicketID string) {
	if projectPath == "" {
		return
	}

	projectCfg, err := config.Load(projectPath)
	if err != nil {
		w.logger.Warn("failed to load project config for queue check", "error", err, "path", projectPath)
		return
	}

	if !projectCfg.Queue {
		return
	}

	queueStore := w.deps.QueueManager.GetStore(projectPath)
	if !queueStore.IsEnabled() {
		return
	}

	store, err := w.deps.StoreManager.GetStore(projectPath)
	if err != nil {
		w.logger.Error("failed to get store for queue spawn", "error", err, "path", projectPath)
		return
	}

	endedTicket, _, err := store.Get(endedTicketID)
	if err != nil {
		w.logger.Warn("failed to get ended ticket for repo lookup", "error", err, "ticket", endedTicketID)
		return
	}

	repo := endedTicket.Repo
	if repo == "" {
		return
	}

	nextTicketID := queueStore.Peek(repo)
	if nextTicketID == "" {
		return
	}

	t, status, err := store.Get(nextTicketID)
	if err != nil {
		w.logger.Warn("queued ticket not found, removing from queue", "ticket", nextTicketID, "error", err)
		_ = queueStore.Remove(repo, nextTicketID)
		w.handleSessionEnded(projectPath, endedTicketID)
		return
	}

	if t.Type != "work" {
		w.logger.Debug("skipping non-work ticket in queue", "ticket", nextTicketID, "type", t.Type)
		_ = queueStore.Remove(repo, nextTicketID)
		w.handleSessionEnded(projectPath, endedTicketID)
		return
	}

	if status != ticket.StatusBacklog {
		w.logger.Debug("queued ticket not in backlog, removing", "ticket", nextTicketID, "status", status)
		_ = queueStore.Remove(repo, nextTicketID)
		w.handleSessionEnded(projectPath, endedTicketID)
		return
	}

	if t.Repo != repo {
		w.logger.Debug("queued ticket has different repo, removing", "ticket", nextTicketID, "expected_repo", repo, "actual_repo", t.Repo)
		_ = queueStore.Remove(repo, nextTicketID)
		w.handleSessionEnded(projectPath, endedTicketID)
		return
	}

	_, err = queueStore.Dequeue(repo)
	if err != nil {
		w.logger.Error("failed to dequeue ticket", "error", err, "ticket", nextTicketID)
		return
	}

	sessionStore := w.deps.SessionManager.GetStore(projectPath)

	result, err := spawn.Orchestrate(w.ctx, spawn.OrchestrateRequest{
		TicketID:      nextTicketID,
		Mode:          "normal",
		ArchitectPath: projectPath,
	}, spawn.OrchestrateDeps{
		Store:        store,
		SessionStore: sessionStore,
		TmuxManager:  w.deps.TmuxManager,
		Logger:       w.logger,
		CortexdPath:  w.deps.CortexdPath,
		DefaultsDir:  w.deps.DefaultsDir,
	})

	if err != nil {
		w.logger.Error("failed to auto-spawn queued ticket", "error", err, "ticket", nextTicketID)
		return
	}

	w.bus.Emit(events.Event{
		Type:          events.TicketDequeued,
		ArchitectPath: projectPath,
		TicketID:      nextTicketID,
		Payload:       map[string]string{"outcome": string(result.Outcome)},
	})

	w.logger.Info("auto-spawned queued ticket", "ticket", nextTicketID, "repo", repo, "outcome", result.Outcome)
}
