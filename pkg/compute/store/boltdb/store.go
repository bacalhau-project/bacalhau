package boltdb

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"time"

	"github.com/rs/zerolog/log"
	bolt "go.etcd.io/bbolt"

	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	"github.com/bacalhau-project/bacalhau/pkg/lib/boltdblib"
	"github.com/bacalhau-project/bacalhau/pkg/lib/marshaller"
	"github.com/bacalhau-project/bacalhau/pkg/lib/watcher"
	boltdb_watcher "github.com/bacalhau-project/bacalhau/pkg/lib/watcher/boltdb"
)

// Store represents an execution store that is backed by a boltdb database
// on disk. The structure of the store is organized into boltdb buckets and sub-buckets
// for efficient data retrieval and organization.
//
// The schema (<key> -> {json-value}) looks like the following, where <> represents
// keys, {} represents values, and undecorated values are boltdb buckets:
//
// * Executions are stored in a bucket called `execution` where each key is
// an execution ID and the value is the JSON representation of the execution.
//
// executions
//
//	|--> <execution-id> -> {store.LocalExecutionState}
//
// * Execution history is stored in a bucket called `execution-history`. Each execution
// that has history is stored in a sub-bucket, whose name is the execution ID.
// Within the execution ID bucket, each key is a sequential number for the
// history item, ensuring they are returned in write order.
//
// execution_history
//
//	|--> <execution-id>
//	          |--> <seqnum> -> {store.LocalStateHistory}
//
// * The job index is stored in a bucket called `idx:executions-by-jobid` where
// each job is represented by a sub-bucket, named after the job ID. Within
// that job bucket, each execution is represented by a key which is the ID of that
// execution, with a nil value.
//
// idx_executions_by_job_id
//
//	|--> <job-id>
//	          |--> <execution-id> -> nil
//
// * The state index is stored in a bucket called `idx:executions-by-state` where
// each execution state is represented by a sub-bucket, named after the state.
// Within each state bucket, executions are indexed by their ID with a nil value.
//
// idx_executions_by_state
//
//	|--> <state>
//	          |--> <execution-id> -> nil
//
// * Additional buckets for event storage:
//   - `events`: Stores event data
//   - `checkpoints`: Stores checkpoint information for the event system
//
// This structure allows for efficient querying of executions by ID, job, and state,
// as well as maintaining a complete history of execution state changes.
type Store struct {
	database   *bolt.DB
	marshaller marshaller.Marshaller
	eventStore *boltdb_watcher.EventStore
}

// NewStore creates a new store backed by a boltdb database at the
// file location provided by the caller.  During initialisation the
// primary buckets are created, but they are not stored in the struct
// as they are tied to the transaction where they are referenced and
// it would mean later transactions will fail unless they obtain their
// own reference to the bucket.
func NewStore(ctx context.Context, dbPath string) (*Store, error) {
	log.Ctx(ctx).Debug().Msgf("creating new bbolt database at %s", dbPath)

	database, err := boltdblib.Open(dbPath)
	if err != nil {
		return nil, err
	}

	err = database.Update(func(tx *bolt.Tx) error {
		buckets := []string{executionsBucket, executionHistoryBucket, idxExecutionsByJobBucket, idxExecutionsByStateBucket}
		for _, b := range buckets {
			_, err = tx.CreateBucketIfNotExists(strToBytes(b))
			if err != nil {
				return fmt.Errorf("error creating bucket %s: %w", b, err)
			}
		}
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error creating database structure: %w", err)
	}

	eventObjectSerializer := watcher.NewJSONSerializer()
	err = eventObjectSerializer.RegisterType("LocalStateHistory", reflect.TypeOf(store.LocalStateHistory{}))
	if err != nil {
		return nil, fmt.Errorf("failed to register LocalStateHistory type: %w", err)
	}
	eventStore, err := boltdb_watcher.NewEventStore(database,
		boltdb_watcher.WithEventsBucket(eventsBucket),
		boltdb_watcher.WithCheckpointBucket(checkpointsBucket),
		boltdb_watcher.WithEventSerializer(eventObjectSerializer),
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create event store: %w", err)
	}

	return &Store{
		database:   database,
		marshaller: marshaller.NewJSONMarshaller(),
		eventStore: eventStore,
	}, nil
}

// GetExecution returns the stored LocalExecutionState structure for the provided execution ID.
func (s *Store) GetExecution(ctx context.Context, executionID string) (store.LocalExecutionState, error) {
	var execution store.LocalExecutionState
	err := s.database.View(func(tx *bolt.Tx) error {
		var err error
		execution, err = s.getExecutionInTx(tx, executionID)
		return err
	})
	return execution, err
}

func (s *Store) getExecutionInTx(tx *bolt.Tx, executionID string) (store.LocalExecutionState, error) {
	var execution store.LocalExecutionState

	data := bucket(tx, executionsBucket).Get(strToBytes(executionID))
	if data == nil {
		return execution, store.NewErrExecutionNotFound(executionID)
	}

	err := s.marshaller.Unmarshal(data, &execution)
	if err != nil {
		return execution, fmt.Errorf("failed to unmarshal execution: %w", err)
	}
	execution.Normalize()
	return execution, nil
}

// GetExecutions retrieves akk if the executions from the job-index bucket for the
// provided Job ID.
func (s *Store) GetExecutions(ctx context.Context, jobID string) ([]store.LocalExecutionState, error) {
	executions := make([]store.LocalExecutionState, 0, defaultSliceRetrievalCapacity)

	err := s.database.View(func(tx *bolt.Tx) error {
		jobBucket := bucket(tx, idxExecutionsByJobBucket, jobID)
		if jobBucket == nil {
			return store.NewErrExecutionsNotFoundForJob(jobID)
		}

		// List all of the keys in the bucket which all have nil values, and are
		// used only as markers to point to the actual execution in the relevant
		// bucket
		return jobBucket.ForEach(func(k, _ []byte) error {
			execution, err := s.getExecutionInTx(tx, string(k))
			if err != nil {
				return err
			}
			executions = append(executions, execution)
			return nil
		})
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

func (s *Store) GetLiveExecutions(ctx context.Context) ([]store.LocalExecutionState, error) {
	executions := make([]store.LocalExecutionState, 0, defaultSliceRetrievalCapacity)

	err := s.database.View(func(tx *bolt.Tx) error {
		for _, state := range store.ExecutionStateTypes() {
			if !state.IsExecuting() {
				continue
			}
			stateBucket := bucket(tx, idxExecutionsByStateBucket, state.String())
			if stateBucket == nil {
				continue
			}
			err := stateBucket.ForEach(func(k, _ []byte) error {
				execution, err := s.getExecutionInTx(tx, string(k))
				if err != nil {
					return err
				}
				executions = append(executions, execution)

				return nil
			})
			if err != nil {
				return err
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

// GetExecutionHistory retrieves the execution history for a single execution
// specified by the executionID parameter.
func (s *Store) GetExecutionHistory(ctx context.Context, executionID string) ([]store.LocalStateHistory, error) {
	history := make([]store.LocalStateHistory, 0, defaultSliceRetrievalCapacity)

	err := s.database.View(func(tx *bolt.Tx) error {
		historyBucket := bucket(tx, executionHistoryBucket, executionID)
		if historyBucket == nil {
			return store.NewErrExecutionHistoryNotFound(executionID)
		}

		// Iterate all of the key-values in the history/executionID bucket as they
		// are the history, and are written in sequential order using the bucket
		// sequence
		return historyBucket.ForEach(func(k, v []byte) error {
			var historyItem store.LocalStateHistory
			err := s.marshaller.Unmarshal(v, &historyItem)
			if err != nil {
				return err
			}

			history = append(history, historyItem)
			return nil
		})
	})

	if err != nil {
		return nil, err
	}
	if len(history) == 0 {
		return nil, store.NewErrExecutionHistoryNotFound(executionID)
	}

	return history, nil
}

func (s *Store) CreateExecution(ctx context.Context, execution store.LocalExecutionState) error {
	execution.Normalize()
	if err := store.ValidateNewExecution(execution); err != nil {
		return fmt.Errorf("CreateExecution failure: %w", err)
	}

	return s.database.Update(func(tx *bolt.Tx) error {
		// Check if execution already exists
		_, err := s.getExecutionInTx(tx, execution.Execution.ID)
		if err == nil {
			return store.NewErrExecutionAlreadyExists(execution.Execution.ID)
		} else if !errors.As(err, &store.ErrExecutionNotFound{}) {
			return err
		}

		// Store execution
		executionData, err := s.marshaller.Marshal(execution)
		if err != nil {
			return err
		}
		if err := bucket(tx, executionsBucket).Put(strToBytes(execution.Execution.ID), executionData); err != nil {
			return err
		}

		// Create job-execution index
		jobBucket, err := bucket(tx, idxExecutionsByJobBucket).CreateBucketIfNotExists(strToBytes(execution.Execution.JobID))
		if err != nil {
			return err
		}
		if err := jobBucket.Put(strToBytes(execution.Execution.ID), nil); err != nil {
			return err
		}

		// Create execution-state index
		stateBucket, err := bucket(tx, idxExecutionsByStateBucket).CreateBucketIfNotExists(strToBytes(execution.State.String()))
		if err != nil {
			return err
		}
		if err := stateBucket.Put(strToBytes(execution.Execution.ID), nil); err != nil {
			return err
		}

		// Append to history
		return s.appendHistory(tx, execution, store.ExecutionStateUndefined, store.NewExecutionMessage)
	})
}

func (s *Store) UpdateExecutionState(ctx context.Context, request store.UpdateExecutionStateRequest) error {
	return s.database.Update(func(tx *bolt.Tx) error {
		execution, err := s.getExecutionInTx(tx, request.ExecutionID)
		if err != nil {
			return err
		}

		if err := request.Validate(execution); err != nil {
			return err
		}

		previousState := execution.State
		execution.State = request.NewState
		execution.Revision++
		execution.UpdateTime = time.Now().UTC()

		if request.RunOutput != nil {
			execution.RunOutput = request.RunOutput
		}

		if request.PublishedResult != nil {
			execution.PublishedResult = request.PublishedResult
		}

		// Update execution
		executionData, err := s.marshaller.Marshal(execution)
		if err != nil {
			return err
		}
		if err = bucket(tx, executionsBucket).Put(strToBytes(execution.Execution.ID), executionData); err != nil {
			return err
		}

		// Update execution-state index
		if err = bucket(tx, idxExecutionsByStateBucket, previousState.String()).Delete(strToBytes(execution.Execution.ID)); err != nil {
			return err
		}
		stateBucket, err := bucket(tx, idxExecutionsByStateBucket).CreateBucketIfNotExists(strToBytes(execution.State.String()))
		if err != nil {
			return err
		}
		if err = stateBucket.Put(strToBytes(execution.Execution.ID), nil); err != nil {
			return err
		}

		// Append to history
		return s.appendHistory(tx, execution, previousState, request.Comment)
	})
}

func (s *Store) appendHistory(tx *bolt.Tx,
	execution store.LocalExecutionState, previousState store.LocalExecutionStateType, comment string) error {
	historyEntry := store.LocalStateHistory{
		ExecutionID:   execution.Execution.ID,
		PreviousState: previousState,
		NewState:      execution.State,
		NewRevision:   execution.Revision,
		Comment:       comment,
		Time:          execution.UpdateTime,
	}
	historyData, err := s.marshaller.Marshal(historyEntry)
	if err != nil {
		return err
	}

	historyBucket, err := bucket(tx, executionHistoryBucket).CreateBucketIfNotExists(strToBytes(execution.Execution.ID))
	if err != nil {
		return err
	}

	seqNum, err := historyBucket.NextSequence()
	if err != nil {
		return err
	}

	err = s.eventStore.StoreEventTx(tx, watcher.OperationCreate, "LocalStateHistory", historyEntry)
	if err != nil {
		return err
	}

	return historyBucket.Put(uint64ToBytes(seqNum), historyData)
}

// DeleteExecution delete the execution, removes its history and removes it
// from the job index (along with the job indexes bucket)
func (s *Store) DeleteExecution(ctx context.Context, executionID string) error {
	return s.database.Update(func(tx *bolt.Tx) error {
		execution, err := s.getExecutionInTx(tx, executionID)
		if err != nil {
			return err
		}

		// Delete execution
		if err := bucket(tx, executionsBucket).Delete(strToBytes(executionID)); err != nil {
			return fmt.Errorf("failed to delete execution: %w", err)
		}

		// Delete from job-execution index
		jobBucket := bucket(tx, idxExecutionsByJobBucket, execution.Execution.JobID)
		if jobBucket == nil {
			log.Ctx(ctx).Warn().Msgf("job bucket not found while deleting execution %s", executionID)
		} else {
			if err = jobBucket.Delete(strToBytes(executionID)); err != nil {
				return fmt.Errorf("failed to delete execution from job index: %w", err)
			}

			// Check if job bucket is empty
			if isEmptyBucket(jobBucket) {
				if err = bucket(tx, idxExecutionsByJobBucket).DeleteBucket(strToBytes(execution.Execution.JobID)); err != nil {
					return fmt.Errorf("failed to delete empty job bucket: %w", err)
				}
			}
		}

		// Delete from execution-state index
		stateBucket := bucket(tx, idxExecutionsByStateBucket, execution.State.String())
		if stateBucket != nil {
			if err := stateBucket.Delete(strToBytes(executionID)); err != nil {
				return fmt.Errorf("failed to delete execution from state index: %w", err)
			}
		}

		// Delete history
		if err = bucket(tx, executionHistoryBucket).DeleteBucket(strToBytes(executionID)); err != nil {
			return fmt.Errorf("failed to delete execution history: %w", err)
		}

		return nil
	})
}

// GetEventStore returns the event store for the execution store
func (s *Store) GetEventStore() watcher.EventStore {
	return s.eventStore
}

func (s *Store) Close(ctx context.Context) error {
	if s.database != nil {
		return s.database.Close()
	}
	return nil
}

func (s *Store) GetExecutionCount(ctx context.Context, state store.LocalExecutionStateType) (uint64, error) {
	var count uint64
	err := s.database.View(func(tx *bolt.Tx) error {
		b := bucket(tx, idxExecutionsByStateBucket, state.String())
		if b == nil {
			return nil
		}

		keyCount := b.Stats().KeyN
		if keyCount < 0 {
			return fmt.Errorf("invalid negative key count from bucket: %d", keyCount)
		}

		count = uint64(keyCount)
		return nil
	})

	return count, err
}

// compile-time check that we implement the interface ExecutionStore
var _ store.ExecutionStore = (*Store)(nil)
