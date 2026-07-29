package store

import (
	"context"
	"encoding/json"
	"sort"
	"sync"

	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/domain"
	"github.com/Liu-ty/ai4se_Coding_Agent_Harness/internal/storeport"
)

// MemoryStore is a concurrency-safe in-process Store implementation.
type MemoryStore struct {
	mu        sync.RWMutex
	runs      map[domain.RunID]domain.Run
	events    map[domain.RunID][]domain.RunEvent
	artifacts map[string]domain.Artifact
	clock     Clock
}

// NewMemory creates an empty in-memory store.
func NewMemory() *MemoryStore {
	return NewMemoryWithClock(realClock{})
}

// NewMemoryWithClock creates an in-memory store with deterministic event time.
func NewMemoryWithClock(clock Clock) *MemoryStore {
	if clock == nil {
		panic("store: nil Clock")
	}
	return &MemoryStore{
		runs:      make(map[domain.RunID]domain.Run),
		events:    make(map[domain.RunID][]domain.RunEvent),
		artifacts: make(map[string]domain.Artifact),
		clock:     clock,
	}
}

func (s *MemoryStore) CreateRun(ctx context.Context, run domain.Run) error {
	if err := validateRunID(run.ID); err != nil {
		return err
	}
	if err := checkContext(ctx); err != nil {
		return err
	}
	run = normalizeRun(run)
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.runs[run.ID]; ok {
		return storeport.ErrRunAlreadyExists
	}
	s.runs[run.ID] = run
	return nil
}

func (s *MemoryStore) CreateRunWithEvent(
	ctx context.Context,
	run domain.Run,
	eventType string,
	payload json.RawMessage,
) (domain.RunEvent, error) {
	if err := validateEvent(run.ID, eventType); err != nil {
		return domain.RunEvent{}, err
	}
	if err := checkContext(ctx); err != nil {
		return domain.RunEvent{}, err
	}
	run = normalizeRun(run)
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.runs[run.ID]; ok {
		return domain.RunEvent{}, storeport.ErrRunAlreadyExists
	}
	s.runs[run.ID] = run
	event := s.appendLocked(run.ID, eventType, payload)
	return cloneEvent(event), nil
}

func (s *MemoryStore) UpdateRun(ctx context.Context, run domain.Run, eventType string, payload json.RawMessage) (domain.RunEvent, error) {
	if err := validateEvent(run.ID, eventType); err != nil {
		return domain.RunEvent{}, err
	}
	if err := checkContext(ctx); err != nil {
		return domain.RunEvent{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.runs[run.ID]; !ok {
		return domain.RunEvent{}, storeport.ErrRunNotFound
	}
	run = normalizeRun(run)
	event := s.appendLocked(run.ID, eventType, payload)
	s.runs[run.ID] = run
	return cloneEvent(event), nil
}

func (s *MemoryStore) UpdateRunIfState(
	ctx context.Context,
	run domain.Run,
	expected domain.RunState,
	eventType string,
	payload json.RawMessage,
) (domain.RunEvent, error) {
	if err := validateEvent(run.ID, eventType); err != nil {
		return domain.RunEvent{}, err
	}
	if err := checkContext(ctx); err != nil {
		return domain.RunEvent{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	current, ok := s.runs[run.ID]
	if !ok {
		return domain.RunEvent{}, storeport.ErrRunNotFound
	}
	if current.State != expected {
		return domain.RunEvent{}, storeport.ErrRunStateChanged
	}
	run = normalizeRun(run)
	event := s.appendLocked(run.ID, eventType, payload)
	s.runs[run.ID] = run
	return cloneEvent(event), nil
}

func (s *MemoryStore) AppendEvent(ctx context.Context, runID domain.RunID, eventType string, payload json.RawMessage) (domain.RunEvent, error) {
	if err := validateEvent(runID, eventType); err != nil {
		return domain.RunEvent{}, err
	}
	if err := checkContext(ctx); err != nil {
		return domain.RunEvent{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.runs[runID]; !ok {
		return domain.RunEvent{}, storeport.ErrRunNotFound
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
	event := newEvent(runID, uint64(len(events)+1), eventType, payload, previousHash, s.clock.Now())
	s.events[runID] = append(events, event)
	return event
}

func (s *MemoryStore) GetRun(ctx context.Context, runID domain.RunID) (domain.Run, error) {
	if err := validateRunID(runID); err != nil {
		return domain.Run{}, err
	}
	if err := checkContext(ctx); err != nil {
		return domain.Run{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	run, ok := s.runs[runID]
	if !ok {
		return domain.Run{}, storeport.ErrRunNotFound
	}
	return run, nil
}

func (s *MemoryStore) ListRuns(ctx context.Context, query storeport.RunListQuery) (storeport.RunPage, error) {
	if query.Limit < 0 || query.Offset < 0 {
		return storeport.RunPage{}, storeport.ErrInvalidRunList
	}
	if err := checkContext(ctx); err != nil {
		return storeport.RunPage{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	runs := make([]domain.Run, 0, len(s.runs))
	var allowed map[domain.RunID]struct{}
	if query.IDs != nil {
		allowed = make(map[domain.RunID]struct{}, len(query.IDs))
		for _, id := range query.IDs {
			allowed[id] = struct{}{}
		}
	}
	for _, run := range s.runs {
		if allowed != nil {
			if _, ok := allowed[run.ID]; !ok {
				continue
			}
		}
		runs = append(runs, run)
	}
	sort.Slice(runs, func(i, j int) bool {
		if runs[i].UpdatedAt.Equal(runs[j].UpdatedAt) {
			return runs[i].ID < runs[j].ID
		}
		return runs[i].UpdatedAt.After(runs[j].UpdatedAt)
	})
	start := query.Offset
	if start > len(runs) {
		start = len(runs)
	}
	end := len(runs)
	hasMore := false
	if query.Limit > 0 && start+query.Limit < end {
		end = start + query.Limit
		hasMore = true
	}
	return storeport.RunPage{Runs: runs[start:end], HasMore: hasMore}, nil
}

func (s *MemoryStore) ListEvents(ctx context.Context, runID domain.RunID, fromSequence uint64) ([]domain.RunEvent, error) {
	if err := validateRunID(runID); err != nil {
		return nil, err
	}
	if err := checkContext(ctx); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, ok := s.runs[runID]; !ok {
		return nil, storeport.ErrRunNotFound
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

func (s *MemoryStore) PutArtifact(ctx context.Context, artifact domain.Artifact) error {
	if err := validateArtifact(artifact); err != nil {
		return err
	}
	if err := checkContext(ctx); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.runs[artifact.RunID]; !ok {
		return storeport.ErrRunNotFound
	}
	s.artifacts[artifact.ID] = cloneArtifact(artifact)
	return nil
}

func (s *MemoryStore) GetArtifact(
	ctx context.Context,
	runID domain.RunID,
	artifactID string,
) (domain.Artifact, error) {
	if err := validateRunID(runID); err != nil {
		return domain.Artifact{}, err
	}
	if artifactID == "" {
		return domain.Artifact{}, storeport.ErrEmptyArtifactID
	}
	if err := checkContext(ctx); err != nil {
		return domain.Artifact{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, ok := s.runs[runID]; !ok {
		return domain.Artifact{}, storeport.ErrRunNotFound
	}
	artifact, ok := s.artifacts[artifactID]
	if !ok || artifact.RunID != runID {
		return domain.Artifact{}, storeport.ErrArtifactNotFound
	}
	return cloneArtifact(artifact), nil
}
