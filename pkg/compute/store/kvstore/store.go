package kvstore

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	"github.com/bacalhau-project/bacalhau/pkg/objectstore"
	"github.com/bacalhau-project/bacalhau/pkg/objectstore/localstore"
	"github.com/rs/zerolog/log"
)

const (
	newExecutionComment = "Execution created"

	PrefixJobs       = "jobs"
	PrefixExecutions = "executions"
	PrefixHistory    = "history"
)

var ExecutionPrefixes = []string{PrefixJobs, PrefixExecutions, PrefixHistory}

type Store struct {
	executions *localstore.Client[ExecutionEnvelope]
	history    *localstore.Client[ExecutionHistoryEnvelope]
	jobs       *localstore.Client[JobIndexEnvelope]
}

func NewStore(ctx context.Context, database *localstore.LocalStore) *Store {
	log.Ctx(ctx).Info().Msg("creating new kvstore")

	return &Store{
		executions: localstore.NewClient[ExecutionEnvelope](ctx, PrefixExecutions, database),
		history:    localstore.NewClient[ExecutionHistoryEnvelope](ctx, PrefixHistory, database),
		jobs:       localstore.NewClient[JobIndexEnvelope](ctx, PrefixJobs, database),
	}
}

func (s *Store) GetExecution(ctx context.Context, id string) (store.Execution, error) {
	log.Ctx(ctx).Debug().
		Str("ExecutionID", id).
		Msg("kvstore.GetExecution")

	envelope, err := s.executions.Get(id)
	if err != nil {
		if errors.Is(err, objectstore.NewErrNotFound(id)) {
			return envelope.Execution, store.NewErrExecutionNotFound(id)
		}
		return envelope.Execution, err
	}
	return envelope.Execution, nil
}

func (s *Store) GetExecutions(ctx context.Context, jobID string) ([]store.Execution, error) {
	log.Ctx(ctx).Debug().
		Str("JobID", jobID).
		Msg("kvstore.GetExecutions")

	identifiers, err := s.jobs.Get(jobID)

	if err != nil || len(identifiers) == 0 {
		return nil, store.NewErrExecutionsNotFoundForJob(jobID)
	}

	// TODO: We can optimize this by fetching en masse rather than one read per
	// execution (if we extend the client impl)
	executions := make([]store.Execution, len(identifiers))
	for i, key := range identifiers {
		execEnv, err := s.executions.Get(key)
		if err != nil {
			return nil, err
		}
		executions[i] = execEnv.Execution
	}

	sort.Slice(executions, func(i, j int) bool {
		return executions[i].UpdateTime.Before(executions[j].UpdateTime)
	})

	return executions, nil
}

func (s *Store) GetExecutionHistory(ctx context.Context, executionID string) ([]store.ExecutionHistory, error) {
	log.Ctx(ctx).Debug().
		Str("ExecutionID", executionID).
		Msg("kvstore.GetExecutionHistory")

	envelope, err := s.history.Get(executionID)
	if err != nil {
		return nil, store.NewErrExecutionHistoryNotFound(executionID)
	}

	return envelope.History, nil
}

func (s *Store) CreateExecution(ctx context.Context, execution store.Execution) error {
	log.Ctx(ctx).Debug().
		Str("ExecutionID", execution.ID).
		Msg("kvstore.CreateExecution")

	_, err := s.executions.Get(execution.ID)
	if err == nil {
		return store.NewErrExecutionAlreadyExists(execution.ID)
	}

	if err := store.ValidateNewExecution(ctx, execution); err != nil {
		return fmt.Errorf("CreateExecution failure: %w", err)
	}

	envelope := ExecutionEnvelope{Execution: execution}
	// Save the execution and we should be indexing it under the job id
	err = s.executions.Put(execution.ID, envelope)
	if err != nil {
		return err
	}

	return s.appendHistory(ctx, execution, store.ExecutionStateUndefined, newExecutionComment)
}

func (s *Store) UpdateExecutionState(ctx context.Context, request store.UpdateExecutionStateRequest) error {
	log.Ctx(ctx).Debug().
		Str("ExecutionID", request.ExecutionID).
		Msg("kvstore.UpdateExecutionState")

	envelope, err := s.executions.Get(request.ExecutionID)
	if err != nil {
		return store.NewErrExecutionNotFound(request.ExecutionID)
	}

	execution := envelope.Execution

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

	err = s.executions.Put(execution.ID, ExecutionEnvelope{Execution: execution})
	if err != nil {
		return err
	}

	return s.appendHistory(ctx, execution, previousState, request.Comment)
}

func (s *Store) appendHistory(
	ctx context.Context,
	updatedExecution store.Execution,
	previousState store.ExecutionState, comment string) error {
	historyEnvelope, err := s.history.Get(updatedExecution.ID)
	if err != nil {
		historyEnvelope = ExecutionHistoryEnvelope{History: make([]store.ExecutionHistory, 0)}
	}

	historyEntry := store.ExecutionHistory{
		ExecutionID:   updatedExecution.ID,
		PreviousState: previousState,
		NewState:      updatedExecution.State,
		NewVersion:    updatedExecution.Version,
		Comment:       comment,
		Time:          updatedExecution.UpdateTime,
	}

	historyEnvelope.History = append(historyEnvelope.History, historyEntry)
	err = s.history.Put(updatedExecution.ID, historyEnvelope)
	if err != nil {
		return err
	}
	return nil
}

func (s *Store) DeleteExecution(ctx context.Context, executionID string) error {
	log.Ctx(ctx).Debug().
		Str("ExecutionID", executionID).
		Msg("kvstore.DeleteExecution")

	envelope, err := s.executions.Get(executionID)
	if err != nil {
		return store.NewErrExecutionNotFound(executionID)
	}

	err = s.executions.Delete(executionID, envelope)
	if err != nil {
		return err
	}

	historyEnvelope, err := s.history.Get(executionID)
	if err != nil {
		return store.NewErrExecutionHistoryNotFound(executionID)
	}

	err = s.history.Delete(executionID, historyEnvelope)
	if err != nil {
		return err
	}
	return nil
}

func (s *Store) GetExecutionCount(ctx context.Context) (uint, error) {
	log.Ctx(ctx).Debug().
		Msg("kvstore.GetExecutionCount")

	// TODO(ross) So, counting based on execution state .... seems like
	// maintaining a prefix for a counter is excessive

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
