package queue

import (
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/kareemaly/cortex/internal/storage"
)

type Store struct {
	path string
	mu   sync.Mutex
}

type RepoQueue struct {
	TicketIDs []string          `json:"ticket_ids"`
	SpawnedAt map[string]string `json:"spawned_at"`
}

type QueueState struct {
	Enabled bool                 `json:"enabled"`
	Queues  map[string]RepoQueue `json:"queues"`
}

func NewStore(path string) *Store {
	return &Store{path: path}
}

func (s *Store) Enqueue(repo, ticketID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	state, err := s.load()
	if err != nil {
		return err
	}

	if !state.Enabled {
		return fmt.Errorf("queue is not enabled")
	}

	if state.Queues == nil {
		state.Queues = make(map[string]RepoQueue)
	}

	queue := state.Queues[repo]
	if queue.TicketIDs == nil {
		queue.TicketIDs = []string{}
	}
	if queue.SpawnedAt == nil {
		queue.SpawnedAt = make(map[string]string)
	}

	for _, id := range queue.TicketIDs {
		if id == ticketID {
			return nil
		}
	}

	queue.TicketIDs = append(queue.TicketIDs, ticketID)
	queue.SpawnedAt[ticketID] = time.Now().UTC().Format(time.RFC3339)
	state.Queues[repo] = queue

	return s.save(state)
}

func (s *Store) Dequeue(repo string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	state, err := s.load()
	if err != nil {
		return "", err
	}

	queue, ok := state.Queues[repo]
	if !ok || len(queue.TicketIDs) == 0 {
		return "", nil
	}

	ticketID := queue.TicketIDs[0]
	queue.TicketIDs = queue.TicketIDs[1:]
	delete(queue.SpawnedAt, ticketID)
	state.Queues[repo] = queue

	if err := s.save(state); err != nil {
		return "", err
	}

	return ticketID, nil
}

func (s *Store) Remove(repo, ticketID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	state, err := s.load()
	if err != nil {
		return err
	}

	queue, ok := state.Queues[repo]
	if !ok {
		return nil
	}

	for i, id := range queue.TicketIDs {
		if id == ticketID {
			queue.TicketIDs = append(queue.TicketIDs[:i], queue.TicketIDs[i+1:]...)
			delete(queue.SpawnedAt, ticketID)
			state.Queues[repo] = queue
			return s.save(state)
		}
	}

	return nil
}

func (s *Store) RemoveFromAllQueues(ticketID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	state, err := s.load()
	if err != nil {
		return err
	}

	for repo, queue := range state.Queues {
		for i, id := range queue.TicketIDs {
			if id == ticketID {
				queue.TicketIDs = append(queue.TicketIDs[:i], queue.TicketIDs[i+1:]...)
				delete(queue.SpawnedAt, ticketID)
				state.Queues[repo] = queue
				break
			}
		}
	}

	return s.save(state)
}

func (s *Store) Peek(repo string) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	state, err := s.load()
	if err != nil {
		return ""
	}

	queue, ok := state.Queues[repo]
	if !ok || len(queue.TicketIDs) == 0 {
		return ""
	}

	return queue.TicketIDs[0]
}

func (s *Store) Position(repo, ticketID string) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	state, err := s.load()
	if err != nil {
		return 0
	}

	queue, ok := state.Queues[repo]
	if !ok {
		return 0
	}

	shortID := storage.ShortID(ticketID)
	for i, id := range queue.TicketIDs {
		if storage.ShortID(id) == shortID {
			return i + 1
		}
	}

	return 0
}

func (s *Store) List(repo string) []string {
	s.mu.Lock()
	defer s.mu.Unlock()

	state, err := s.load()
	if err != nil {
		return nil
	}

	queue, ok := state.Queues[repo]
	if !ok {
		return nil
	}

	result := make([]string, len(queue.TicketIDs))
	copy(result, queue.TicketIDs)
	return result
}

func (s *Store) IsEnabled() bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	state, err := s.load()
	if err != nil {
		return false
	}

	return state.Enabled
}

func (s *Store) SetEnabled(enabled bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	state, err := s.load()
	if err != nil {
		return err
	}

	state.Enabled = enabled
	return s.save(state)
}

func (s *Store) GetTicketRepo(ticketID string) string {
	s.mu.Lock()
	defer s.mu.Unlock()

	state, err := s.load()
	if err != nil {
		return ""
	}

	shortID := storage.ShortID(ticketID)
	for repo, queue := range state.Queues {
		for _, id := range queue.TicketIDs {
			if storage.ShortID(id) == shortID {
				return repo
			}
		}
	}

	return ""
}

func (s *Store) load() (*QueueState, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return &QueueState{
				Enabled: false,
				Queues:  make(map[string]RepoQueue),
			}, nil
		}
		return nil, fmt.Errorf("read queue file: %w", err)
	}

	if len(data) == 0 {
		return &QueueState{
			Enabled: false,
			Queues:  make(map[string]RepoQueue),
		}, nil
	}

	var state QueueState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("unmarshal queue: %w", err)
	}

	if state.Queues == nil {
		state.Queues = make(map[string]RepoQueue)
	}

	return &state, nil
}

func (s *Store) save(state *QueueState) error {
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal queue: %w", err)
	}

	return storage.AtomicWriteFile(s.path, data)
}
