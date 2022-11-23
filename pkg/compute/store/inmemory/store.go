package inmemory

import (
	"context"
	"fmt"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/compute/store"
	sync "github.com/lukemarsden/golang-mutex-tracer"
)

type InMemoryStore struct {
	executionMap map[string]*store.Execution
	shardMap     map[string][]string
	history      map[string][]*store.ExecutionHistory
	mu           sync.RWMutex
}

func NewInMemoryStore() *InMemoryStore {
	res := &InMemoryStore{
		executionMap: make(map[string]*store.Execution),
		shardMap:     make(map[string][]string),
		history:      make(map[string][]*store.ExecutionHistory),
	}
	res.mu.EnableTracerWithOpts(sync.Opts{
		Threshold: 10 * time.Millisecond,
		Id:        "InMemoryExecutionStore.mu",
	})
	return res
}

func (s *InMemoryStore) GetExecution(ctx context.Context, id string) (*store.Execution, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.executionMap[id], nil
}

func (s *InMemoryStore) GetExecutions(ctx context.Context, shardID string) ([]*store.Execution, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	executionIDs := s.shardMap[shardID]
	executions := make([]*store.Execution, len(executionIDs))
	for i, id := range executionIDs {
		executions[i] = s.executionMap[id]
	}
	return executions, nil
}

func (s *InMemoryStore) CreateExecution(ctx context.Context, execution *store.Execution) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.executionMap[execution.ID]; ok {
		return store.NewErrExecutionAlreadyExists(execution.ID)
	}
	if err := store.ValidateNewExecution(ctx, execution); err != nil {
		return fmt.Errorf("could not create invalid execution: %w", err)
	}

	s.executionMap[execution.ID] = execution
	s.shardMap[execution.Shard.ID()] = append(s.shardMap[execution.Shard.ID()], execution.ID)
	return nil
}

func (s *InMemoryStore) UpdateExecutionState(ctx context.Context, request store.UpdateExecutionStateRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	execution, ok := s.executionMap[request.ExecutionID]
	if !ok {
		return store.NewErrExecutionNotFound(request.ExecutionID)
	}
	if request.ExpectedState != store.ExecutionStateUndefined && execution.State != request.ExpectedState {
		return store.NewErrInvalidExecutionState(request.ExecutionID, execution.State, request.ExpectedState)
	}
	if request.ExpectedVersion != 0 && execution.Version != request.ExpectedVersion {
		return store.NewErrInvalidExecutionVersion(request.ExecutionID, execution.Version, request.ExpectedVersion)
	}
	historyEntry := &store.ExecutionHistory{
		ExecutionID:   execution.ID,
		PreviousState: execution.State,
		NewState:      request.NewState,
		NewVersion:    execution.Version + 1,
		Comment:       request.Comment,
	}
	s.history[execution.ID] = append(s.history[execution.ID], historyEntry)
	execution.State = historyEntry.NewState
	execution.Version = historyEntry.NewVersion
	return nil
}

func (s *InMemoryStore) DeleteExecution(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	execution, ok := s.executionMap[id]
	if ok {
		delete(s.executionMap, id)
		delete(s.history, id)
		shardID := execution.Shard.ID()
		shard := s.shardMap[shardID]
		for i, e := range shard {
			if e == id {
				s.shardMap[shardID] = append(shard[:i], shard[i+1:]...)
				break
			}
		}
	}
	return nil
}

// compile-time check that we implement the interface ExecutionStore
var _ store.ExecutionStore = (*InMemoryStore)(nil)
