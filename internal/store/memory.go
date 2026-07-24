package store

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/domain"
)

// MemoryStore is a concurrency-safe in-process Store implementation.
type MemoryStore struct {
	mu        sync.RWMutex
	runs      map[domain.RunID]domain.Run
	events    map[domain.RunID][]domain.RunEvent
	artifacts map[string]domain.Artifact
}

// NewMemory creates an empty in-memory store.
func NewMemory() *MemoryStore {
	return &MemoryStore{
		runs:      make(map[domain.RunID]domain.Run),
		events:    make(map[domain.RunID][]domain.RunEvent),
		artifacts: make(map[string]domain.Artifact),
	}
}

func (s *MemoryStore) CreateRun(_ context.Context, run domain.Run) error {
	if err := validateRunID(run.ID); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.runs[run.ID] = run
	return nil
}

func (s *MemoryStore) UpdateRun(_ context.Context, run domain.Run, eventType string, payload json.RawMessage) (domain.RunEvent, error) {
	if err := validateEvent(run.ID, eventType); err != nil {
		return domain.RunEvent{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.runs[run.ID]; !ok {
		return domain.RunEvent{}, ErrRunNotFound
	}
	event := s.appendLocked(run.ID, eventType, payload)
	s.runs[run.ID] = run
	return cloneEvent(event), nil
}

func (s *MemoryStore) AppendEvent(_ context.Context, runID domain.RunID, eventType string, payload json.RawMessage) (domain.RunEvent, error) {
	if err := validateEvent(runID, eventType); err != nil {
		return domain.RunEvent{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.runs[runID]; !ok {
		return domain.RunEvent{}, ErrRunNotFound
	}
	event := s.appendLocked(runID, eventType, payload)
	return cloneEvent(event), nil
}

func (s *MemoryStore) appendLocked(runID domain.RunID, eventType string, payload json.RawMessage) domain.RunEvent {
	events := s.events[runID]
	previousHash := ""
	if len(events) > 0 {
		previousHash = events[len(events)-1].Hash
	}
	event := newEvent(runID, uint64(len(events)+1), eventType, payload, previousHash, time.Now())
	s.events[runID] = append(events, event)
	return event
}

func (s *MemoryStore) GetRun(_ context.Context, runID domain.RunID) (domain.Run, error) {
	if err := validateRunID(runID); err != nil {
		return domain.Run{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	run, ok := s.runs[runID]
	if !ok {
		return domain.Run{}, ErrRunNotFound
	}
	return run, nil
}

func (s *MemoryStore) ListEvents(_ context.Context, runID domain.RunID, fromSequence uint64) ([]domain.RunEvent, error) {
	if err := validateRunID(runID); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, ok := s.runs[runID]; !ok {
		return nil, ErrRunNotFound
	}
	stored := s.events[runID]
	events := make([]domain.RunEvent, 0, len(stored))
	for _, event := range stored {
		if event.Sequence >= fromSequence {
			events = append(events, cloneEvent(event))
		}
	}
	return events, nil
}

func (s *MemoryStore) PutArtifact(_ context.Context, artifact domain.Artifact) error {
	if err := validateArtifact(artifact); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.runs[artifact.RunID]; !ok {
		return ErrRunNotFound
	}
	s.artifacts[artifact.ID] = cloneArtifact(artifact)
	return nil
}
