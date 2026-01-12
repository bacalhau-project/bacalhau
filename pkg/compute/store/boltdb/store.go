package boltdb

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"sort"
	"time"

	"github.com/imdario/mergo"
	"github.com/rs/zerolog/log"
	bolt "go.etcd.io/bbolt"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	"github.com/bacalhau-project/bacalhau/pkg/lib/boltdblib"
	"github.com/bacalhau-project/bacalhau/pkg/lib/marshaller"
	"github.com/bacalhau-project/bacalhau/pkg/lib/watcher"
	boltdb_watcher "github.com/bacalhau-project/bacalhau/pkg/lib/watcher/boltdb"
	"github.com/bacalhau-project/bacalhau/pkg/models"
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
//	|--> <execution-id> -> {*models.Execution}
//
// * Execution events is stored in a bucket called `execution_events`. Each execution
// that has events is stored in a sub-bucket, whose name is the execution ID.
// Within the execution ID bucket, each key is a sequential number for the
// event item, ensuring they are returned in write order.
//
// execution_events
//
//	|--> <execution-id>
//	          |--> <seqnum> -> {models.Event}
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
// as well as maintaining a complete events of execution state changes.
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
		buckets := []string{
			executionsBucket,
			executionEventsBucket,
			idxExecutionsByJobBucket,
			idxExecutionsByStateBucket,
			sequenceTrackingBucket}
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
	err = errors.Join(
		eventObjectSerializer.RegisterType(compute.EventObjectExecutionUpsert, reflect.TypeOf(models.ExecutionUpsert{})),
		eventObjectSerializer.RegisterType(compute.EventObjectExecutionEvent, reflect.TypeOf(models.Event{})),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to register event object types: %w", err)
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

// BeginTx starts a new writable transaction for the store
func (s *Store) BeginTx(ctx context.Context) (boltdblib.TxContext, error) {
	tx, err := s.database.Begin(true)
	if err != nil {
		return nil, err
	}
	return boltdblib.NewTxContext(ctx, tx), nil
}

// GetExecution returns the stored Execution structure for the provided execution ID.
func (s *Store) GetExecution(ctx context.Context, executionID string) (*models.Execution, error) {
	var execution *models.Execution
	err := s.database.View(func(tx *bolt.Tx) error {
		var err error
		execution, err = s.getExecutionInTx(tx, executionID)
		return err
	})
	return execution, err
}

func (s *Store) getExecutionInTx(tx *bolt.Tx, executionID string) (*models.Execution, error) {
	var execution *models.Execution

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
func (s *Store) GetExecutions(ctx context.Context, jobID string) ([]*models.Execution, error) {
	executions := make([]*models.Execution, 0, defaultSliceRetrievalCapacity)

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
		return executions[i].ModifyTime > executions[j].ModifyTime
	})

	return executions, nil
}

func (s *Store) GetLiveExecutions(ctx context.Context) ([]*models.Execution, error) {
	executions := make([]*models.Execution, 0, defaultSliceRetrievalCapacity)

	err := s.database.View(func(tx *bolt.Tx) error {
		for _, state := range models.ExecutionStateTypes() {
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
		return executions[i].ModifyTime > executions[j].ModifyTime
	})

	return executions, nil
}

// GetExecutionEvents retrieves the execution events for a single execution
// specified by the executionID parameter.
func (s *Store) GetExecutionEvents(ctx context.Context, executionID string) ([]*models.Event, error) {
	events := make([]*models.Event, 0, defaultSliceRetrievalCapacity)

	err := s.database.View(func(tx *bolt.Tx) error {
		bu := bucket(tx, executionEventsBucket, executionID)
		if bu == nil {
			return store.NewErrExecutionEventsNotFound(executionID)
		}

		return bu.ForEach(func(k, v []byte) error {
			event := new(models.Event)
			err := s.marshaller.Unmarshal(v, event)
			if err != nil {
				return err
			}

			events = append(events, event)
			return nil
		})
	})

	if err != nil {
		return nil, err
	}
	if len(events) == 0 {
		return nil, store.NewErrExecutionEventsNotFound(executionID)
	}

	return events, nil
}

func (s *Store) AddExecutionEvent(ctx context.Context, executionID string, events ...*models.Event) error {
	return s.database.Update(func(tx *bolt.Tx) error {
		_, err := s.getExecutionInTx(tx, executionID)
		if err != nil {
			return err
		}

		return s.addExecutionEventInTx(tx, executionID, events)
	})
}

func (s *Store) addExecutionEventInTx(tx *bolt.Tx, executionID string, events []*models.Event) error {
	for i := range events {
		event := events[i]
		eventData, err := s.marshaller.Marshal(event)
		if err != nil {
			return err
		}

		bkt, err := bucket(tx, executionEventsBucket).CreateBucketIfNotExists(strToBytes(executionID))
		if err != nil {
			return err
		}

		seqNum, err := bkt.NextSequence()
		if err != nil {
			return err
		}

		if err = bkt.Put(uint64ToBytes(seqNum), eventData); err != nil {
			return err
		}

		if err = s.eventStore.StoreEventTx(tx, watcher.StoreEventRequest{
			Operation:  watcher.OperationCreate,
			ObjectType: compute.EventObjectExecutionEvent,
			Object:     event,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) CreateExecution(ctx context.Context, execution models.Execution, events ...*models.Event) error {
	execution.Normalize()
	err := store.ValidateNewExecution(&execution)
	if err != nil {
		return err
	}

	return s.database.Update(func(tx *bolt.Tx) error {
		execution.CreateTime = time.Now().UTC().UnixNano()
		execution.ModifyTime = execution.CreateTime
		execution.Revision = 1

		// Check if execution already exists
		_, err = s.getExecutionInTx(tx, execution.ID)
		if err == nil {
			return store.NewErrExecutionAlreadyExists(execution.ID)
		} else if !errors.As(err, &store.ErrExecutionNotFound{}) {
			return err
		}

		// Store execution
		executionData, err := s.marshaller.Marshal(execution)
		if err != nil {
			return err
		}
		if err := bucket(tx, executionsBucket).Put(strToBytes(execution.ID), executionData); err != nil {
			return err
		}

		// Create job-execution index
		jobBucket, err := bucket(tx, idxExecutionsByJobBucket).CreateBucketIfNotExists(strToBytes(execution.JobID))
		if err != nil {
			return err
		}
		if err := jobBucket.Put(strToBytes(execution.ID), nil); err != nil {
			return err
		}

		// Create execution-state index
		stateBucket, err := bucket(tx, idxExecutionsByStateBucket).
			CreateBucketIfNotExists(strToBytes(execution.ComputeState.StateType.String()))
		if err != nil {
			return err
		}
		if err := stateBucket.Put(strToBytes(execution.ID), nil); err != nil {
			return err
		}

		// Store events
		if err = s.addExecutionEventInTx(tx, execution.ID, events); err != nil {
			return err
		}

		return s.eventStore.StoreEventTx(tx, watcher.StoreEventRequest{
			Operation:  watcher.OperationCreate,
			ObjectType: compute.EventObjectExecutionUpsert,
			Object:     models.ExecutionUpsert{Current: &execution, Events: events},
		})
	})
}

func (s *Store) UpdateExecutionState(ctx context.Context, request store.UpdateExecutionRequest) error {
	return s.database.Update(func(tx *bolt.Tx) error {
		existingExecution, err := s.getExecutionInTx(tx, request.ExecutionID)
		if err != nil {
			return err
		}

		// check the expected state
		if err = request.Condition.Validate(existingExecution); err != nil {
			return err
		}
		if existingExecution.IsTerminalComputeState() {
			return store.NewErrExecutionAlreadyTerminal(
				request.ExecutionID, existingExecution.ComputeState.StateType, request.NewValues.ComputeState.StateType)
		}

		// populate default values, maintain existing execution createTime
		newExecution := request.NewValues
		newExecution.CreateTime = existingExecution.CreateTime
		newExecution.ModifyTime = time.Now().UTC().UnixNano()
		newExecution.Revision = existingExecution.Revision + 1
		newExecution.Normalize()

		err = mergo.Merge(&newExecution, existingExecution)
		if err != nil {
			return fmt.Errorf("failed to merge execution values: %w", err)
		}
		// Update execution
		executionData, err := s.marshaller.Marshal(newExecution)
		if err != nil {
			return err
		}
		if err = bucket(tx, executionsBucket).Put(strToBytes(newExecution.ID), executionData); err != nil {
			return err
		}

		// Update execution-state index
		if err = bucket(tx, idxExecutionsByStateBucket, stateBucketKeyStr(existingExecution)).
			Delete(strToBytes(existingExecution.ID)); err != nil {
			return err
		}
		stateBucket, err := bucket(tx, idxExecutionsByStateBucket).
			CreateBucketIfNotExists(stateBucketKey(&newExecution))
		if err != nil {
			return err
		}
		if err = stateBucket.Put(strToBytes(newExecution.ID), nil); err != nil {
			return err
		}

		// Store events
		if err = s.addExecutionEventInTx(tx, newExecution.ID, request.Events); err != nil {
			return err
		}

		return s.eventStore.StoreEventTx(tx, watcher.StoreEventRequest{
			Operation:  watcher.OperationUpdate,
			ObjectType: compute.EventObjectExecutionUpsert,
			Object: models.ExecutionUpsert{
				Current: &newExecution, Previous: existingExecution, Events: request.Events,
			},
		})
	})
}

// DeleteExecution delete the execution, removes its events and removes it
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
		jobBucket := bucket(tx, idxExecutionsByJobBucket, execution.JobID)
		if jobBucket == nil {
			log.Ctx(ctx).Warn().Msgf("job bucket not found while deleting execution %s", executionID)
		} else {
			if err = jobBucket.Delete(strToBytes(executionID)); err != nil {
				return fmt.Errorf("failed to delete execution from job index: %w", err)
			}

			// Check if job bucket is empty
			if isEmptyBucket(jobBucket) {
				if err = bucket(tx, idxExecutionsByJobBucket).DeleteBucket(strToBytes(execution.JobID)); err != nil {
					return fmt.Errorf("failed to delete empty job bucket: %w", err)
				}
			}
		}

		// Delete from execution-state index
		stateBucket := bucket(tx, idxExecutionsByStateBucket, stateBucketKeyStr(execution))
		if stateBucket != nil {
			if err := stateBucket.Delete(strToBytes(executionID)); err != nil {
				return fmt.Errorf("failed to delete execution from state index: %w", err)
			}
		}

		// Delete execution events, if any
		err = bucket(tx, executionEventsBucket).DeleteBucket(strToBytes(executionID))
		if err != nil && !errors.Is(err, bolt.ErrBucketNotFound) { //nolint:staticcheck // TODO: migrate to bbolt/errors package
			return fmt.Errorf("failed to delete execution events: %w", err)
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

func (s *Store) GetExecutionCount(ctx context.Context, state models.ExecutionStateType) (uint64, error) {
	var count uint64
	err := s.database.View(func(tx *bolt.Tx) error {
		b := bucket(tx, idxExecutionsByStateBucket, state.String())
		if b == nil {
			return nil
		}

		count = uint64(b.Stats().KeyN) //nolint:gosec
		return nil
	})

	return count, err
}

func (s *Store) Checkpoint(ctx context.Context, name string, sequenceNumber uint64) error {
	if name == "" {
		return store.NewErrCheckpointNameBlank()
	}
	return s.database.Update(func(tx *bolt.Tx) error {
		b := bucket(tx, sequenceTrackingBucket)
		if b == nil {
			return fmt.Errorf("sequence tracking bucket not found")
		}

		// Store sequence number as bytes
		err := b.Put(strToBytes(name), uint64ToBytes(sequenceNumber))
		if err != nil {
			return fmt.Errorf("store sequence number: %w", err)
		}

		return nil
	})
}

func (s *Store) GetCheckpoint(ctx context.Context, name string) (uint64, error) {
	if name == "" {
		return 0, store.NewErrCheckpointNameBlank()
	}
	var seq uint64
	err := s.database.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(strToBytes(sequenceTrackingBucket))
		if b == nil {
			return nil // Return 0 if bucket doesn't exist yet
		}

		data := b.Get(strToBytes(name))
		if data == nil {
			return nil // Return 0 if sequence doesn't exist
		}

		seq = bytesToUint64(data)
		return nil
	})
	if err != nil {
		return 0, fmt.Errorf("get sequence tracking value: %w", err)
	}

	return seq, nil
}

// compile-time check that we implement the interface ExecutionStore
var _ store.ExecutionStore = (*Store)(nil)
