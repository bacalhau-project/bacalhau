package badgerdb

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"time"

	"github.com/dgraph-io/badger/v3"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	"github.com/bacalhau-project/bacalhau/pkg/lib/marshaller"
	"github.com/bacalhau-project/bacalhau/pkg/lib/validate"
)

const (
	defaultSliceRetrievalCapacity = 10
)

// Key prefixes for different types of data
const (
	executionPrefix         = "execution:"
	executionsByJobPrefix   = "idx:executions_by_jobid:"
	executionHistoryPrefix  = "execution_history:"
	executionsByStatePrefix = "idx:executions_by_state:"
)

// Store represents an execution store that is backed by a BadgerDB database.
type Store struct {
	database   *badger.DB
	marshaller marshaller.Marshaller
}

// NewStore creates a new store backed by a BadgerDB database at the
// file location provided by the caller.
func NewStore(dbPath string) (*Store, error) {
	log.Debug().Msgf("creating new BadgerDB store at %s", dbPath)

	err := errors.Join(
		validate.IsDirectory(dbPath, "BadgerDB path is not a directory: %s", dbPath),
		validate.IsWritable(dbPath, "BadgerDB path is not writable: %s", dbPath),
	)
	if err != nil {
		return nil, err
	}

	opts := badger.DefaultOptions(dbPath)
	opts.Logger = newBadgerLoggerAdapter(log.Logger, zerolog.WarnLevel)
	db, err := badger.Open(opts)
	if err != nil {
		return nil, fmt.Errorf("error opening BadgerDB: %w", err)
	}

	return &Store{
		database:   db,
		marshaller: marshaller.NewJSONMarshaller(),
	}, nil
}

// Helper methods for key generation
func executionKey(executionID string) []byte {
	return []byte(executionPrefix + executionID)
}

func executionByJobKey(jobID string) []byte {
	return []byte(executionsByJobPrefix + jobID + ":")
}

func executionHistoryKey(executionID string) []byte {
	return []byte(executionHistoryPrefix + executionID + ":")
}

func executionByStateKey(state store.LocalExecutionStateType) []byte {
	return []byte(executionsByStatePrefix + state.String() + ":")
}

// GetExecution retrieves a single execution by its ID.
func (s *Store) GetExecution(ctx context.Context, executionID string) (store.LocalExecutionState, error) {
	var execution store.LocalExecutionState
	err := s.database.View(func(txn *badger.Txn) error {
		var err error
		execution, err = s.getExecutionInTxn(txn, executionID)
		return err
	})
	if err != nil {
		return store.LocalExecutionState{}, err
	}
	return execution, nil
}

// GetExecutions retrieves all executions for a given job ID.
func (s *Store) GetExecutions(ctx context.Context, jobID string) ([]store.LocalExecutionState, error) {
	executions := make([]store.LocalExecutionState, 0, defaultSliceRetrievalCapacity)

	err := s.database.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()

		prefix := executionByJobKey(jobID)
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			executionID := string(it.Item().Key()[len(prefix):])
			execution, err := s.getExecutionInTxn(txn, executionID)
			if err != nil {
				return err
			}
			executions = append(executions, execution)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(executions) == 0 {
		return nil, store.NewErrExecutionsNotFoundForJob(jobID)
	}
	sort.Slice(executions, func(i, j int) bool {
		return executions[i].UpdateTime.Before(executions[j].UpdateTime)
	})
	return executions, nil
}

// getExecutionInTxn is a helper function to get an execution within a transaction.
func (s *Store) getExecutionInTxn(txn *badger.Txn, executionID string) (store.LocalExecutionState, error) {
	var execution store.LocalExecutionState
	item, err := txn.Get(executionKey(executionID))
	if err != nil {
		if errors.Is(err, badger.ErrKeyNotFound) {
			return execution, store.NewErrExecutionNotFound(executionID)
		}
		return execution, err
	}
	err = item.Value(func(val []byte) error {
		return s.marshaller.Unmarshal(val, &execution)
	})
	if err != nil {
		return execution, err
	}
	execution.Normalize()
	return execution, nil
}

// GetLiveExecutions retrieves all executions in an active state.
func (s *Store) GetLiveExecutions(ctx context.Context) ([]store.LocalExecutionState, error) {
	executions := make([]store.LocalExecutionState, 0, defaultSliceRetrievalCapacity)
	err := s.database.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()

		for _, state := range store.ExecutionStateTypes() {
			if !state.IsExecuting() {
				continue
			}
			prefix := executionByStateKey(state)
			for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
				executionID := string(it.Item().Key()[len(prefix):])
				execution, err := s.getExecutionInTxn(txn, executionID)
				if err != nil {
					return err
				}
				executions = append(executions, execution)
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(executions, func(i, j int) bool {
		return executions[i].UpdateTime.Before(executions[j].UpdateTime)
	})

	return executions, nil
}

// GetExecutionHistory retrieves the history of a single execution.
func (s *Store) GetExecutionHistory(ctx context.Context, executionID string) ([]store.LocalStateHistory, error) {
	history := make([]store.LocalStateHistory, 0, defaultSliceRetrievalCapacity)

	err := s.database.View(func(txn *badger.Txn) error {
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()

		prefix := executionHistoryKey(executionID)
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			var historyItem store.LocalStateHistory
			err := it.Item().Value(func(val []byte) error {
				return s.marshaller.Unmarshal(val, &historyItem)
			})
			if err != nil {
				return err
			}
			history = append(history, historyItem)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(history) == 0 {
		return nil, store.NewErrExecutionHistoryNotFound(executionID)
	}
	return history, nil
}

// CreateExecution creates a new execution in the database.
func (s *Store) CreateExecution(ctx context.Context, execution store.LocalExecutionState) error {
	execution.Normalize()
	if err := store.ValidateNewExecution(execution); err != nil {
		return fmt.Errorf("CreateExecution failure: %w", err)
	}

	return s.database.Update(func(txn *badger.Txn) error {
		// Check if execution already exists
		_, err := txn.Get(executionKey(execution.Execution.ID))
		if err == nil {
			return store.NewErrExecutionAlreadyExists(execution.Execution.ID)
		} else if !errors.Is(err, badger.ErrKeyNotFound) {
			return err
		}

		// Store execution
		executionData, err := s.marshaller.Marshal(execution)
		if err != nil {
			return err
		}
		if err := txn.Set(executionKey(execution.Execution.ID), executionData); err != nil {
			return err
		}

		// Create job-execution index
		if err := txn.Set(append(executionByJobKey(execution.Execution.JobID), execution.Execution.ID...), nil); err != nil {
			return err
		}

		// Create execution-state index
		if err := txn.Set(append(executionByStateKey(execution.State), execution.Execution.ID...), nil); err != nil {
			return err
		}

		// Append to history
		if err := s.appendHistory(txn, execution, store.ExecutionStateUndefined, store.NewExecutionMessage); err != nil {
			return err
		}

		return nil
	})
}

// UpdateExecutionState updates the state of an execution.
func (s *Store) UpdateExecutionState(ctx context.Context, request store.UpdateExecutionStateRequest) error {
	return s.database.Update(func(txn *badger.Txn) error {
		execution, err := s.getExecutionInTxn(txn, request.ExecutionID)
		if err != nil {
			return err
		}

		if err := request.Validate(execution); err != nil {
			return err
		}

		previousState := execution.State
		execution.State = request.NewState
		execution.Version++
		execution.UpdateTime = time.Now()

		// Update execution
		executionData, err := s.marshaller.Marshal(execution)
		if err != nil {
			return err
		}
		if err := txn.Set(executionKey(execution.Execution.ID), executionData); err != nil {
			return err
		}

		// Update execution-state index
		if err := txn.Delete(append(executionByStateKey(previousState), execution.Execution.ID...)); err != nil {
			return err
		}
		if err := txn.Set(append(executionByStateKey(execution.State), execution.Execution.ID...), nil); err != nil {
			return err
		}

		// Append to history
		if err := s.appendHistory(txn, execution, previousState, request.Comment); err != nil {
			return err
		}
		return nil
	})
}

// appendHistory adds a new history entry for an execution.
func (s *Store) appendHistory(txn *badger.Txn,
	execution store.LocalExecutionState, previousState store.LocalExecutionStateType, comment string) error {
	historyEntry := store.LocalStateHistory{
		ExecutionID:   execution.Execution.ID,
		PreviousState: previousState,
		NewState:      execution.State,
		NewVersion:    execution.Version,
		Comment:       comment,
		Time:          execution.UpdateTime,
	}
	historyData, err := s.marshaller.Marshal(historyEntry)
	if err != nil {
		return err
	}

	// Use UpdateTime's nanoseconds as the sequence number
	seqNum := uint64(execution.UpdateTime.UnixNano())

	key := fmt.Sprintf("%s%020d", executionHistoryKey(execution.Execution.ID), seqNum)
	return txn.Set([]byte(key), historyData)
}

// DeleteExecution removes an execution and its associated data from the database.
func (s *Store) DeleteExecution(ctx context.Context, executionID string) error {
	return s.database.Update(func(txn *badger.Txn) error {
		execution, err := s.getExecutionInTxn(txn, executionID)
		if err != nil {
			if errors.Is(err, badger.ErrKeyNotFound) {
				return store.NewErrExecutionNotFound(executionID)
			}
			return err
		}

		// Delete execution
		if err := txn.Delete(executionKey(executionID)); err != nil {
			return err
		}

		// Delete from job-execution index
		if err := txn.Delete(append(executionByJobKey(execution.Execution.JobID), executionID...)); err != nil {
			return err
		}

		// Delete from execution-state index
		if err := txn.Delete(append(executionByStateKey(execution.State), executionID...)); err != nil {
			return err
		}

		// Delete history
		it := txn.NewIterator(badger.DefaultIteratorOptions)
		defer it.Close()
		prefix := executionHistoryKey(executionID)
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			if err := txn.Delete(it.Item().Key()); err != nil {
				return err
			}
		}

		return nil
	})
}

// Close closes the BadgerDB database.
func (s *Store) Close(ctx context.Context) error {
	return s.database.Close()
}

// GetExecutionCount returns the count of executions in a specific state.
func (s *Store) GetExecutionCount(ctx context.Context, state store.LocalExecutionStateType) (uint64, error) {
	var count uint64
	err := s.database.View(func(txn *badger.Txn) error {
		opts := badger.DefaultIteratorOptions
		opts.PrefetchValues = false // We only need keys, not values
		it := txn.NewIterator(opts)
		defer it.Close()

		prefix := executionByStateKey(state)
		for it.Seek(prefix); it.ValidForPrefix(prefix); it.Next() {
			count++
		}
		return nil
	})
	return count, err
}

// Ensure Store implements the ExecutionStore interface
var _ store.ExecutionStore = (*Store)(nil)
