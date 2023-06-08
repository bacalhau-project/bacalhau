package persistent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	"github.com/bacalhau-project/bacalhau/pkg/objectstore"
	"github.com/bacalhau-project/bacalhau/pkg/objectstore/commands"
	"github.com/bacalhau-project/bacalhau/pkg/objectstore/local"
	sync "github.com/bacalhau-project/golang-mutex-tracer"
)

const (
	newExecutionComment    = "Execution created"
	PrefixExecution        = "executions"
	PrefixExecutionHistory = "execution-history"
	PrefixJobExecutions    = "job-executions"
)

type Store struct {
	db objectstore.ObjectStore
	mu sync.RWMutex
}

func NewStore() (*Store, error) {
	db, err := objectstore.GetImplementation(
		objectstore.LocalImplementation,
		local.WithPrefixes(PrefixExecution, PrefixExecutionHistory, PrefixJobExecutions),
	)
	if err != nil {
		return nil, err
	}

	db.CallbackHooks().RegisterUpdate(store.Execution{}, updateJobExecutionList)
	db.CallbackHooks().RegisterDelete(store.Execution{}, deleteJobExecutionList)

	res := &Store{db: db}
	res.mu.EnableTracerWithOpts(sync.Opts{
		Threshold: 10 * time.Millisecond,
		Id:        "InMemoryExecutionStore.mu",
	})
	return res, nil
}

func updateJobExecutionList(object any) ([]commands.Command, error) {
	execution, ok := object.(store.Execution)
	if !ok {
		return nil, fmt.Errorf("callback type did not match: got %T", object)
	}

	return []commands.Command{
		// Add the execution ID to the list of IDs found at PrefixJobExecutions/jobID
		commands.NewCommand(PrefixJobExecutions, execution.Job.ID(), commands.AddToSet(execution.ID)),
	}, nil
}

func deleteJobExecutionList(object any) ([]commands.Command, error) {
	execution, ok := object.(store.Execution)
	if !ok {
		return nil, fmt.Errorf("callback type did not match: got %T", object)
	}

	return []commands.Command{
		// Add the execution ID to the list of IDs found at PrefixJobExecutions/jobID
		commands.NewCommand(PrefixJobExecutions, execution.Job.ID(), commands.DeleteFromSet(execution.ID)),
	}, nil
}

func (s *Store) GetExecution(ctx context.Context, id string) (store.Execution, error) {
	var execution store.Execution

	bytes, err := s.db.Get(ctx, PrefixExecution, id)
	if err != nil {
		return execution, err
	}
	if bytes == nil {
		return execution, store.NewErrExecutionNotFound(id)
	}

	err = json.Unmarshal(bytes, &execution)
	if err != nil {
		return execution, err
	}

	return execution, nil
}

// TODO
func (s *Store) GetExecutions(ctx context.Context, jobID string) ([]store.Execution, error) {
	var execList []string
	execListBytes, err := s.db.Get(ctx, PrefixJobExecutions, jobID)
	if err != nil {
		return nil, err
	}

	if execListBytes == nil {
		return nil, store.NewErrExecutionsNotFoundForJob(jobID)
	}

	err = json.Unmarshal(execListBytes, &execList)
	if err != nil {
		return nil, err
	}

	if len(execList) == 0 {
		return nil, store.NewErrExecutionsNotFoundForJob(jobID)
	}

	// TODO: We need GetBatch() so we can fetch multiples of same prefix
	executions := make([]store.Execution, len(execList))
	for i, execID := range execList {
		var execution store.Execution
		ebytes, _ := s.db.Get(ctx, PrefixExecution, execID)
		_ = json.Unmarshal(ebytes, &execution)
		executions[i] = execution
	}

	// Sort by CreateTime so that we get them back in the order we stored them
	sort.Slice(executions, func(i, j int) bool {
		return executions[i].CreateTime.Before(executions[j].CreateTime)
	})

	return executions, nil
}

func (s *Store) GetExecutionHistory(ctx context.Context, id string) ([]store.ExecutionHistory, error) {
	var history []store.ExecutionHistory

	bytes, err := s.db.Get(ctx, PrefixExecutionHistory, id)
	if err != nil {
		return history, err
	}
	if bytes == nil {
		return history, store.NewErrExecutionHistoryNotFound(id)
	}

	err = json.Unmarshal(bytes, &history)
	if err != nil {
		return history, err
	}
	return history, nil
}

func (s *Store) CreateExecution(ctx context.Context, execution store.Execution) error {
	_, err := s.GetExecution(ctx, execution.ID)
	if !errors.Is(err, store.ErrExecutionNotFound{ExecutionID: execution.ID}) {
		return store.NewErrExecutionAlreadyExists(execution.ID)
	}

	if err := store.ValidateNewExecution(ctx, execution); err != nil {
		return fmt.Errorf("CreateExecution failure: %w", err)
	}

	err = s.db.Put(ctx, PrefixExecution, execution.ID, execution)
	if err != nil {
		return err
	}

	err = s.appendHistory(ctx, execution, store.ExecutionStateUndefined, newExecutionComment)
	if err != nil {
		return err
	}

	// TODO
	// from job-id -> list of executions
	// s.jobMap[execution.Job.ID()] = append(s.jobMap[execution.Job.ID()], execution.ID)

	return nil
}

func (s *Store) UpdateExecutionState(ctx context.Context, request store.UpdateExecutionStateRequest) error {
	execution, err := s.GetExecution(ctx, request.ExecutionID)
	if err != nil {
		return err
	}

	if execution.ID == "" {
		return store.NewErrExecutionNotFound(request.ExecutionID)
	}

	if request.ExpectedState != store.ExecutionStateUndefined && execution.State != request.ExpectedState {
		return store.NewErrInvalidExecutionState(request.ExecutionID, execution.State, request.ExpectedState)
	}

	if request.ExpectedVersion != 0 && execution.Version != request.ExpectedVersion {
		return store.NewErrInvalidExecutionVersion(request.ExecutionID, execution.Version, request.ExpectedVersion)
	}

	if execution.State.IsTerminal() {
		return store.NewErrExecutionAlreadyTerminal(request.ExecutionID, execution.State, request.NewState)
	}

	previousState := execution.State
	execution.State = request.NewState
	execution.Version += 1
	execution.UpdateTime = time.Now()

	err = s.db.Put(ctx, PrefixExecution, request.ExecutionID, execution)
	if err != nil {
		return err
	}

	return s.appendHistory(ctx, execution, previousState, request.Comment)
}

// Adds a new execution history item to the list of items for the provided execution,
// creating the list if necessary.
func (s *Store) appendHistory(
	ctx context.Context,
	updatedExecution store.Execution,
	previousState store.ExecutionState,
	comment string) error {
	historyEntry := store.ExecutionHistory{
		ExecutionID:   updatedExecution.ID,
		PreviousState: previousState,
		NewState:      updatedExecution.State,
		NewVersion:    updatedExecution.Version,
		Comment:       comment,
		Time:          updatedExecution.UpdateTime,
	}

	var items []store.ExecutionHistory

	// Get the current list of history items for this ID
	lst, err := s.db.Get(ctx, PrefixExecutionHistory, updatedExecution.ID)
	if err != nil {
		return err
	}
	if lst != nil {
		err = json.Unmarshal(lst, &items)
		if err != nil {
			return err
		}
	}

	items = append(items, historyEntry)
	return s.db.Put(ctx, PrefixExecutionHistory, updatedExecution.ID, items)
}

// TODO: CLeanup
func (s *Store) DeleteExecution(ctx context.Context, id string) error {
	execution, err := s.GetExecution(ctx, id)
	if err != nil {
		return err
	}

	// Delete execution and execution history
	_ = s.db.Delete(ctx, PrefixExecution, id, execution)
	_ = s.db.Delete(ctx, PrefixExecutionHistory, id, store.ExecutionHistory{})

	// Rely on Deletion trigger to remove the execution from the `PrefixJobExecutions`
	// and then check if it is the empty list, at which point we should delete it

	var execList []string
	execListBytes, _ := s.db.Get(ctx, PrefixJobExecutions, execution.Job.ID())
	_ = json.Unmarshal(execListBytes, &execList)
	if len(execList) == 0 {
		// TODO: The remove tasks shoulds auto-cleanup
		_ = s.db.Delete(ctx, PrefixJobExecutions, execution.Job.ID(), []string{})
	}

	return nil
}

// TODO
func (s *Store) GetExecutionCount(ctx context.Context) (uint, error) {
	// var counter uint
	// for _, execution := range s.executionMap {
	// 	if execution.State == store.ExecutionStateCompleted {
	// 		counter++
	// 	}
	// }
	// return counter, nil
	return 0, nil
}

// compile-time check that we implement the interface ExecutionStore
var _ store.ExecutionStore = (*Store)(nil)
