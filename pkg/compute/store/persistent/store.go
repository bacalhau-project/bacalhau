package persistent

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	"github.com/bacalhau-project/bacalhau/pkg/objectstore"
	"github.com/bacalhau-project/bacalhau/pkg/objectstore/index"
	"github.com/bacalhau-project/bacalhau/pkg/objectstore/local"
	sync "github.com/bacalhau-project/golang-mutex-tracer"
	"github.com/rs/zerolog/log"
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

func NewStore(ctx context.Context, nodeID string) (*Store, error) {
	var filepath string

	// Target folder
	if nodeID == "" {
		// Set the filepath to empty for a test database
		filepath = ""
	} else {
		filepath = "/tmp/bacalhau/executions/" + strings.ToLower(nodeID)
	}

	db, err := objectstore.GetImplementation(
		ctx,
		objectstore.LocalImplementation,
		local.WithPrefixes(PrefixExecution, PrefixExecutionHistory, PrefixJobExecutions),
		local.WithDataFile(filepath),
	)
	if err != nil {
		return nil, err
	}

	log.Debug().Str("NodeID", nodeID).Msgf("creating local objectstore in %s", filepath)

	db.CallbackHooks().RegisterUpdate(PrefixExecution, updateJobExecutionList)
	db.CallbackHooks().RegisterDelete(PrefixExecution, deleteJobExecutionList)

	res := &Store{db: db}
	res.mu.EnableTracerWithOpts(sync.Opts{
		Threshold: 10 * time.Millisecond,
		Id:        "InMemoryExecutionStore.mu",
	})
	return res, nil
}

func updateJobExecutionList(object any) ([]index.IndexCommand, error) {
	execution, ok := object.(store.Execution)
	if !ok {
		return nil, fmt.Errorf("callback type did not match: got %T", object)
	}

	return []index.IndexCommand{
		// Add the execution ID to the list of IDs found at PrefixJobExecutions/jobID
		index.NewIndexCommand(PrefixJobExecutions, execution.Job.ID(), index.AddToSet(execution.ID)),
	}, nil
}

func deleteJobExecutionList(object any) ([]index.IndexCommand, error) {
	execution, ok := object.(store.Execution)
	if !ok {
		return nil, fmt.Errorf("callback type did not match: got %T", object)
	}

	return []index.IndexCommand{
		// Add the execution ID to the list of IDs found at PrefixJobExecutions/jobID
		index.NewIndexCommand(PrefixJobExecutions, execution.Job.ID(), index.DeleteFromSet(execution.ID)),
	}, nil
}

func (s *Store) GetExecution(ctx context.Context, id string) (store.Execution, error) {
	var execution store.Execution

	err := s.db.Get(ctx, PrefixExecution, id, &execution)
	if err != nil {
		return execution, store.NewErrExecutionNotFound(id)
	}

	return execution, nil
}

// TODO
func (s *Store) GetExecutions(ctx context.Context, jobID string) ([]store.Execution, error) {
	var execList []string
	err := s.db.Get(ctx, PrefixJobExecutions, jobID, &execList)
	if err != nil {
		return nil, store.NewErrExecutionsNotFoundForJob(jobID)
	}

	if len(execList) == 0 {
		return nil, store.NewErrExecutionsNotFoundForJob(jobID)
	}

	var executions []store.Execution
	_ = s.db.GetBatch(ctx, PrefixExecution, execList, &executions)

	// Sort by CreateTime so that we get them back in the order we stored them
	sort.Slice(executions, func(i, j int) bool {
		return executions[i].CreateTime.Before(executions[j].CreateTime)
	})

	return executions, nil
}

func (s *Store) GetExecutionHistory(ctx context.Context, id string) ([]store.ExecutionHistory, error) {
	var history []store.ExecutionHistory

	err := s.db.Get(ctx, PrefixExecutionHistory, id, &history)
	if err != nil {
		return history, store.NewErrExecutionHistoryNotFound(id)
	}
	if len(history) == 0 {
		return history, store.NewErrExecutionHistoryNotFound(id)
	}

	return history, nil
}

func (s *Store) CreateExecution(ctx context.Context, execution store.Execution) error {
	exec, err := s.GetExecution(ctx, execution.ID)
	if err == nil && exec.ID != "" {
		// We didn't get an error, which means we found the thing we wanted
		return store.ErrExecutionNotFound{ExecutionID: execution.ID}
	}

	if err := store.ValidateNewExecution(ctx, execution); err != nil {
		return fmt.Errorf("CreateExecution failure: %w", err)
	}

	err = s.db.Put(ctx, PrefixExecution, execution.ID, execution)
	if err != nil {
		return err
	}

	err = s.appendHistory(ctx, execution, store.ExecutionStateUndefined, newExecutionComment, true)
	if err != nil {
		return err
	}

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

	return s.appendHistory(ctx, execution, previousState, request.Comment, false)
}

// Adds a new execution history item to the list of items for the provided execution,
// creating the list if necessary.
func (s *Store) appendHistory(
	ctx context.Context,
	updatedExecution store.Execution,
	previousState store.ExecutionState,
	comment string,
	first bool) error {
	historyEntry := store.ExecutionHistory{
		ExecutionID:   updatedExecution.ID,
		PreviousState: previousState,
		NewState:      updatedExecution.State,
		NewVersion:    updatedExecution.Version,
		Comment:       comment,
		Time:          updatedExecution.UpdateTime,
	}

	var items []store.ExecutionHistory

	if !first {
		// Get the existing history
		err := s.db.Get(ctx, PrefixExecutionHistory, updatedExecution.ID, &items)
		if err != nil {
			return fmt.Errorf("no history found for execution: %s", updatedExecution.ID)
		}

		if len(items) == 0 {
			return fmt.Errorf("no history found for execution: %s", updatedExecution.ID)
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
	_ = s.db.Get(ctx, PrefixJobExecutions, execution.Job.ID(), &execList)
	if len(execList) == 0 {
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

func (s *Store) Close(ctx context.Context) error {
	return s.db.Close(ctx)
}

// compile-time check that we implement the interface ExecutionStore
var _ store.ExecutionStore = (*Store)(nil)
