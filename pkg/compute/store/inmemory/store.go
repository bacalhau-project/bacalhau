package inmemory

import (
	"context"
	"fmt"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/compute/store"
	sync "github.com/lukemarsden/golang-mutex-tracer"
)

type Store struct {
	executionMap map[string]store.Execution
	shardMap     map[string][]string
	history      map[string][]store.ExecutionHistory
	mu           sync.RWMutex
}

func NewStore() *Store {
	res := &Store{
		executionMap: make(map[string]store.Execution),
		shardMap:     make(map[string][]string),
		history:      make(map[string][]store.ExecutionHistory),
	}
	res.mu.EnableTracerWithOpts(sync.Opts{
		Threshold: 10 * time.Millisecond,
		Id:        "InMemoryExecutionStore.mu",
	})
	return res
}

func (s *Store) GetExecution(ctx context.Context, id string) (store.Execution, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	execution, ok := s.executionMap[id]
	if !ok {
		return execution, store.NewErrExecutionNotFound(id)
	}
	return execution, nil
}

func (s *Store) GetExecutions(ctx context.Context, shardID string) ([]store.Execution, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	executionIDs, ok := s.shardMap[shardID]
	if !ok {
		return []store.Execution{}, store.NewErrExecutionsNotFound(shardID)
	}
	executions := make([]store.Execution, len(executionIDs))
	for i, id := range executionIDs {
		executions[i] = s.executionMap[id]
	}
	return executions, nil
}

func (s *Store) GetExecutionHistory(ctx context.Context, id string) ([]store.ExecutionHistory, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	history, ok := s.history[id]
	if !ok {
		return history, store.NewErrExecutionHistoryNotFound(id)
	}
	return history, nil
}

func (s *Store) CreateExecution(ctx context.Context, execution store.Execution) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.executionMap[execution.ID]; ok {
		return store.NewErrExecutionAlreadyExists(execution.ID)
	}
	if err := store.ValidateNewExecution(ctx, execution); err != nil {
		return fmt.Errorf("CreateExecution failure: %w", err)
	}

	s.executionMap[execution.ID] = execution
	s.shardMap[execution.Shard.ID()] = append(s.shardMap[execution.Shard.ID()], execution.ID)
	return nil
}

func (s *Store) UpdateExecutionState(ctx context.Context, request store.UpdateExecutionStateRequest) error {
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
	historyEntry := store.ExecutionHistory{
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

func (s *Store) DeleteExecution(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	execution, ok := s.executionMap[id]
	if ok {
		delete(s.executionMap, id)
		delete(s.history, id)
		shardID := execution.Shard.ID()
		shardExecutions := s.shardMap[shardID]
		if len(shardExecutions) == 1 {
			delete(s.shardMap, shardID)
		} else {
			for i, executionID := range shardExecutions {
				if executionID == id {
					s.shardMap[shardID] = append(shardExecutions[:i], shardExecutions[i+1:]...)
					break
				}
			}
		}
	}
	return nil
}

// compile-time check that we implement the interface ExecutionStore
var _ store.ExecutionStore = (*Store)(nil)
