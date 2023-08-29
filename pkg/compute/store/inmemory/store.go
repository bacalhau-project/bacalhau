package inmemory

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	sync "github.com/bacalhau-project/golang-mutex-tracer"
	"golang.org/x/exp/maps"
)

const newExecutionComment = "LocalExecutionState created"

type Store struct {
	executionMap map[string]store.LocalExecutionState
	jobMap       map[string][]string
	liveMap      map[string]struct{}
	history      map[string][]store.LocalStateHistory
	mu           sync.RWMutex
}

func NewStore() *Store {
	res := &Store{
		executionMap: make(map[string]store.LocalExecutionState),
		jobMap:       make(map[string][]string),
		liveMap:      make(map[string]struct{}),
		history:      make(map[string][]store.LocalStateHistory),
	}
	res.mu.EnableTracerWithOpts(sync.Opts{
		Threshold: 10 * time.Millisecond,
		Id:        "InMemoryExecutionStore.mu",
	})
	return res
}

func (s *Store) GetExecution(ctx context.Context, id string) (store.LocalExecutionState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	execution, ok := s.executionMap[id]
	if !ok {
		return execution, store.NewErrExecutionNotFound(id)
	}
	return execution, nil
}

func (s *Store) GetExecutions(ctx context.Context, jobID string) ([]store.LocalExecutionState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	executionIDs, ok := s.jobMap[jobID]
	if !ok {
		return []store.LocalExecutionState{}, store.NewErrExecutionsNotFoundForJob(jobID)
	}
	executions := make([]store.LocalExecutionState, len(executionIDs))
	for i, id := range executionIDs {
		executions[i] = s.executionMap[id]
	}
	return executions, nil
}

func (s *Store) GetLiveExecutions(ctx context.Context) ([]store.LocalExecutionState, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	liveIDs := maps.Keys(s.liveMap)

	executions := make([]store.LocalExecutionState, len(liveIDs))
	for i, id := range liveIDs {
		executions[i] = s.executionMap[id]
	}

	// Ensure executions are returned oldest first
	sort.Slice(executions, func(i, j int) bool {
		return executions[i].UpdateTime.Before(executions[j].UpdateTime)
	})

	return executions, nil
}

func (s *Store) GetExecutionHistory(ctx context.Context, id string) ([]store.LocalStateHistory, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	history, ok := s.history[id]
	if !ok {
		return history, store.NewErrExecutionHistoryNotFound(id)
	}
	return history, nil
}

func (s *Store) CreateExecution(ctx context.Context, localExecutionState store.LocalExecutionState) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	execution := localExecutionState.Execution
	if _, ok := s.executionMap[execution.ID]; ok {
		return store.NewErrExecutionAlreadyExists(execution.ID)
	}
	if err := store.ValidateNewExecution(localExecutionState); err != nil {
		return fmt.Errorf("CreateExecution failure: %w", err)
	}

	s.executionMap[execution.ID] = localExecutionState
	s.jobMap[execution.JobID] = append(s.jobMap[execution.JobID], execution.ID)
	s.appendHistory(localExecutionState, store.ExecutionStateUndefined, newExecutionComment)
	return nil
}

func (s *Store) UpdateExecutionState(ctx context.Context, request store.UpdateExecutionStateRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	localExecutionState, ok := s.executionMap[request.ExecutionID]
	if !ok {
		return store.NewErrExecutionNotFound(request.ExecutionID)
	}
	if request.ExpectedState != store.ExecutionStateUndefined && localExecutionState.State != request.ExpectedState {
		return store.NewErrInvalidExecutionState(request.ExecutionID, localExecutionState.State, request.ExpectedState)
	}
	if request.ExpectedVersion != 0 && localExecutionState.Version != request.ExpectedVersion {
		return store.NewErrInvalidExecutionVersion(request.ExecutionID, localExecutionState.Version, request.ExpectedVersion)
	}
	if localExecutionState.State.IsTerminal() {
		return store.NewErrExecutionAlreadyTerminal(request.ExecutionID, localExecutionState.State, request.NewState)
	}

	previousState := localExecutionState.State
	localExecutionState.State = request.NewState
	localExecutionState.Version += 1
	localExecutionState.UpdateTime = time.Now()
	s.executionMap[localExecutionState.Execution.ID] = localExecutionState
	s.appendHistory(localExecutionState, previousState, request.Comment)

	if localExecutionState.State.IsExecuting() {
		s.liveMap[localExecutionState.Execution.ID] = struct{}{}
	} else {
		delete(s.liveMap, localExecutionState.Execution.ID)
	}

	return nil
}

func (s *Store) appendHistory(updatedExecution store.LocalExecutionState, previousState store.LocalExecutionStateType, comment string) {
	historyEntry := store.LocalStateHistory{
		ExecutionID:   updatedExecution.Execution.ID,
		PreviousState: previousState,
		NewState:      updatedExecution.State,
		NewVersion:    updatedExecution.Version,
		Comment:       comment,
		Time:          updatedExecution.UpdateTime,
	}
	s.history[updatedExecution.Execution.ID] = append(s.history[updatedExecution.Execution.ID], historyEntry)
}

func (s *Store) DeleteExecution(ctx context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	localExecutionState, ok := s.executionMap[id]
	if ok {
		delete(s.executionMap, id)
		delete(s.history, id)
		jobID := localExecutionState.Execution.JobID
		jobExecutions := s.jobMap[jobID]
		if len(jobExecutions) == 1 {
			delete(s.jobMap, jobID)
		} else {
			for i, executionID := range jobExecutions {
				if executionID == id {
					s.jobMap[jobID] = append(jobExecutions[:i], jobExecutions[i+1:]...)
					break
				}
			}
		}
	}
	return nil
}

func (s *Store) GetExecutionCount(ctx context.Context, state store.LocalExecutionStateType) (uint64, error) {
	var counter uint64
	for _, execution := range s.executionMap {
		if execution.State == state {
			counter++
		}
	}
	return counter, nil
}

func (s *Store) Close(ctx context.Context) error {
	return nil
}

// compile-time check that we implement the interface ExecutionStore
var _ store.ExecutionStore = (*Store)(nil)
