package boltdb

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	"github.com/bacalhau-project/bacalhau/pkg/lib/marshaller"
	"github.com/rs/zerolog/log"
	bolt "go.etcd.io/bbolt"
)

const (
	newExecutionComment = "LocalExecutionState created"

	DefaultSliceRetrievalCapacity = 10

	BucketExecutionsName = "execution"
	BucketHistoryName    = "execution-history"
	BucketJobIndexName   = "execution-index"
	BucketLiveIndexName  = "execution-live-index"
)

// Store represents an execution store that is backed by a boltdb database
// on disk.  The structure of the store is in boltdb buckets and sub-buckets
// so that they can be created and retrieved easily.
//
// The schema (<key> {json-value}) looks like the following where <> represents
// keys, {} represents values and undecorated values are boltdb buckets.
//
// * Executions are stored in a bucket called `executions` where each key is
// an execution ID and the value is the JSON representation.
//
// execution
//     |--> <execution-id> -> {store.LocalExecutionState}
//
// * LocalExecutionState history is stored in a bucket called `history`. Each execution
// that has history is stored in a sub-bucket, whose name is the execution ID.
// Within the execution id bucket, each key is a sequential value for the
// history item being written so they are returned in write-order
//
// execution-history
//     |--> execution-id
//               |-> <seqnum> -> {store.LocalStateHistory}
//
// * The job index is stored in a bucket called execution-index where
// each job is represented by a new bucket, named after the job ID.  Within
// that job, each execution is represented by a key which is the id of that
// execution, the value being nil.
//
// execution-index
//     |--> job-id
//               |-> <execution-id> -> {}
//
// * The live index is stored in a bucket called execution-live-index where each
// execution that is in an active state (currently ExecutionStateBidAccepted).
// This is used at node start to check what nodes _should_ be running.
//
// execution-live-index
//    |-> <execution-id> -> {}
//

type Store struct {
	database   *bolt.DB
	marshaller marshaller.Marshaller

	starting     sync.WaitGroup
	stateCounter *StateCounter
}

// NewStore creates a new store backed by a boltdb database at the
// file location provided by the caller.  During initialisation the
// primary buckets are created, but they are not stored in the struct
// as they are tied to the transaction where they are referenced and
// it would mean later transactions will fail unless they obtain their
// own reference to the bucket.
func NewStore(ctx context.Context, dbPath string) (*Store, error) {
	store := &Store{
		marshaller:   marshaller.NewJSONMarshaller(),
		starting:     sync.WaitGroup{},
		stateCounter: NewStateCounter(),
	}
	log.Ctx(ctx).Info().Msgf("creating new bbolt database at %s", dbPath)

	database, err := GetDatabase(dbPath)
	if err != nil {
		if err == bolt.ErrTimeout {
			return nil, fmt.Errorf("timed out while opening database, is file %s in use?", dbPath)
		}
		return nil, err
	}

	store.database = database
	err = store.database.Update(func(tx *bolt.Tx) error {
		_, err = tx.CreateBucketIfNotExists([]byte(BucketExecutionsName))
		if err != nil {
			return err
		}

		_, err = tx.CreateBucketIfNotExists([]byte(BucketHistoryName))
		if err != nil {
			return err
		}

		_, err = tx.CreateBucketIfNotExists([]byte(BucketJobIndexName))
		if err != nil {
			return err
		}

		_, err = tx.CreateBucketIfNotExists([]byte(BucketLiveIndexName))
		if err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("error creating database structure: %s", err)
	}

	// Populate the state counter for the
	store.starting.Add(1)
	go store.populateStateCounter(ctx)

	return store, nil
}

// getExecutionsBucket helper gets a reference to the executions bucket within
// the supplied transaction.
func (s *Store) getExecutionsBucket(tx *bolt.Tx) *bolt.Bucket {
	return tx.Bucket([]byte(BucketExecutionsName))
}

// getHistoryBucket helper gets a reference to the execution history bucket
// within the supplied transaction.
func (s *Store) getHistoryBucket(tx *bolt.Tx) *bolt.Bucket {
	return tx.Bucket([]byte(BucketHistoryName))
}

// getJobIndexBucket helper gets a reference to the job index bucket within
// the supplied transaction.
func (s *Store) getJobIndexBucket(tx *bolt.Tx) *bolt.Bucket {
	return tx.Bucket([]byte(BucketJobIndexName))
}

// getLiveIndexBucket helper gets a reference to the live execution index bucket
func (s *Store) getLiveIndexBucket(tx *bolt.Tx) *bolt.Bucket {
	return tx.Bucket([]byte(BucketLiveIndexName))
}

// GetExecution returns the stored LocalExecutionState structure for the provided execution ID.
func (s *Store) GetExecution(ctx context.Context, executionID string) (store.LocalExecutionState, error) {
	log.Ctx(ctx).Trace().
		Str("ExecutionID", executionID).
		Msg("boltdb.GetExecution")

	var execution store.LocalExecutionState
	err := s.database.View(func(tx *bolt.Tx) (err error) {
		execution, err = s.getExecution(tx, executionID)
		return
	})

	execution.Normalize()
	return execution, err
}

func (s *Store) getExecution(tx *bolt.Tx, executionID string) (store.LocalExecutionState, error) {
	var execution store.LocalExecutionState

	executionsBucket := tx.Bucket([]byte(BucketExecutionsName))
	data := executionsBucket.Get([]byte(executionID))
	if data == nil {
		return execution, store.NewErrExecutionNotFound(executionID)
	}

	err := s.marshaller.Unmarshal(data, &execution)
	return execution, err
}

// GetExecutions retrieves akk if the executions from the job-index bucket for the
// provided Job ID.
func (s *Store) GetExecutions(ctx context.Context, jobID string) ([]store.LocalExecutionState, error) {
	log.Ctx(ctx).Trace().
		Str("JobID", jobID).
		Msg("boltdb.GetExecutions")

	var executions []store.LocalExecutionState
	err := s.database.View(func(tx *bolt.Tx) (err error) {
		executions, err = s.getExecutions(tx, jobID)
		return
	})
	if err != nil || len(executions) == 0 {
		return nil, store.NewErrExecutionsNotFoundForJob(jobID)
	}

	sort.Slice(executions, func(i, j int) bool {
		return executions[i].UpdateTime.Before(executions[j].UpdateTime)
	})

	for _, execution := range executions {
		execution.Normalize()
	}
	return executions, nil
}

func (s *Store) getExecutions(tx *bolt.Tx, jobID string) ([]store.LocalExecutionState, error) {
	jobIndexBucket := s.getJobIndexBucket(tx)
	jobBucket := jobIndexBucket.Bucket([]byte(jobID))
	if jobBucket == nil {
		return nil, store.NewErrJobNotFound(jobID)
	}

	executions := make([]store.LocalExecutionState, 0, DefaultSliceRetrievalCapacity)

	// List all of the keys in the bucket which all have nil values, and are
	// used only as markers to point to the actual execution in the relevant
	// bucket
	err := jobBucket.ForEach(func(key []byte, _ []byte) error {
		execution, err := s.getExecution(tx, string(key))
		if err != nil {
			return err
		}
		executions = append(executions, execution)

		return nil
	})

	return executions, err
}

func (s *Store) GetLiveExecutions(ctx context.Context) ([]store.LocalExecutionState, error) {
	log.Ctx(ctx).Trace().
		Msg("boltdb.GetLiveExecutions")

	var executions []store.LocalExecutionState
	err := s.database.View(func(tx *bolt.Tx) (err error) {
		executions, err = s.getLiveExecutions(tx)
		return
	})
	if err != nil {
		return nil, err
	}

	sort.Slice(executions, func(i, j int) bool {
		return executions[i].UpdateTime.Before(executions[j].UpdateTime)
	})

	return executions, nil
}

func (s *Store) getLiveExecutions(tx *bolt.Tx) ([]store.LocalExecutionState, error) {
	liveIndexBkt := s.getLiveIndexBucket(tx)

	executions := make([]store.LocalExecutionState, 0, DefaultSliceRetrievalCapacity)

	// List all of the keys in the live index bucket, and fetch the appropriate
	// executions
	err := liveIndexBkt.ForEach(func(key []byte, _ []byte) error {
		execution, err := s.getExecution(tx, string(key))
		if err != nil {
			return err
		}
		executions = append(executions, execution)

		return nil
	})
	return executions, err
}

// GetExecutionHistory retrieves the execution history for a single execution
// specified by the executionID parameter.
func (s *Store) GetExecutionHistory(ctx context.Context, executionID string) ([]store.LocalStateHistory, error) {
	log.Ctx(ctx).Trace().
		Str("ExecutionID", executionID).
		Msg("boltdb.GetExecutionHistory")

	var history []store.LocalStateHistory
	err := s.database.View(func(tx *bolt.Tx) (err error) {
		history, err = s.getExecutionHistory(tx, executionID)
		return
	})
	if err != nil || len(history) == 0 {
		return nil, store.NewErrExecutionHistoryNotFound(executionID)
	}

	return history, err
}

func (s *Store) getExecutionHistory(tx *bolt.Tx, executionID string) ([]store.LocalStateHistory, error) {
	historyBucket := s.getHistoryBucket(tx)

	executionBucket := historyBucket.Bucket([]byte(executionID))
	if executionBucket == nil {
		return nil, store.NewErrExecutionNotFound(executionID)
	}

	history := make([]store.LocalStateHistory, 0, DefaultSliceRetrievalCapacity)

	// Iterate all of the key-values in the history/executionID bucket as they
	// are the history, and are written in sequential order using the bucket
	// sequence
	err := executionBucket.ForEach(func(key []byte, data []byte) error {
		var item store.LocalStateHistory

		err := s.marshaller.Unmarshal(data, &item)
		if err != nil {
			return err
		}

		history = append(history, item)
		return nil
	})

	return history, err
}

// CreateExecution creates a new execution in the database and also sets up the
// relevant index entry for the new execution.
func (s *Store) CreateExecution(ctx context.Context, localExecutionState store.LocalExecutionState) error {
	log.Ctx(ctx).Trace().
		Str("ExecutionID", localExecutionState.Execution.ID).
		Msg("boltdb.CreateExecution")

	localExecutionState.Normalize()
	if err := store.ValidateNewExecution(localExecutionState); err != nil {
		return fmt.Errorf("CreateExecution failure: %w", err)
	}

	return s.database.Update(func(tx *bolt.Tx) (err error) {
		err = s.createExecution(tx, localExecutionState)
		if err == nil {
			// If we are confident that the value was written without error
			// and we won't rollback
			s.stateCounter.IncrementState(localExecutionState.State, 1)
		}
		return
	})
}

func (s *Store) createExecution(tx *bolt.Tx, localExecutionState store.LocalExecutionState) error {
	_, err := s.getExecution(tx, localExecutionState.Execution.ID)
	if err == nil { // deliberate, we require an err to continue
		return store.NewErrExecutionAlreadyExists(localExecutionState.Execution.ID)
	}

	// Write the execution to the executions bucket
	executionData, err := s.marshaller.Marshal(localExecutionState)
	if err != nil {
		return err
	}

	executionsBucket := s.getExecutionsBucket(tx)
	err = executionsBucket.Put([]byte(localExecutionState.Execution.ID), executionData)
	if err != nil {
		return err
	}

	// Create the job bucket in the job index if it does not already exist
	jobIndexBucket := s.getJobIndexBucket(tx)
	jobBucket, err := jobIndexBucket.CreateBucketIfNotExists([]byte(localExecutionState.Execution.JobID))
	if err != nil {
		return err
	}

	err = jobBucket.Put([]byte(localExecutionState.Execution.ID), nil)
	if err != nil {
		return err
	}

	return s.appendHistory(tx, localExecutionState, store.ExecutionStateUndefined, newExecutionComment)
}

// UpdateExecutionState updates the current state of the execution
func (s *Store) UpdateExecutionState(ctx context.Context, request store.UpdateExecutionStateRequest) error {
	log.Ctx(ctx).Trace().
		Str("ExecutionID", request.ExecutionID).
		Msg("boltdb.UpdateExecutionState")

	return s.database.Update(func(tx *bolt.Tx) (err error) {
		previousState, err := s.updateExecutionState(tx, request)
		if err == nil {
			s.stateCounter.DecrementState(previousState, 1)
			s.stateCounter.IncrementState(request.NewState, 1)
		}
		return err
	})
}

func (s *Store) updateExecutionState(tx *bolt.Tx, request store.UpdateExecutionStateRequest) (store.LocalExecutionStateType, error) {
	emptyState := store.ExecutionStateUndefined

	localExecutionState, err := s.getExecution(tx, request.ExecutionID)
	if err != nil {
		return emptyState, store.NewErrExecutionNotFound(request.ExecutionID)
	}

	if err := request.Validate(localExecutionState); err != nil {
		return emptyState, err
	}

	if localExecutionState.State.IsTerminal() {
		return emptyState, store.NewErrExecutionAlreadyTerminal(request.ExecutionID, localExecutionState.State, request.NewState)
	}

	previousState := localExecutionState.State
	localExecutionState.State = request.NewState
	localExecutionState.Version += 1
	localExecutionState.UpdateTime = time.Now()

	// Having modified the execution, we're going to write it back over the top of the previous entry
	// before appending a copy of the history to the history bucket

	data, err := s.marshaller.Marshal(localExecutionState)
	if err != nil {
		return emptyState, err
	}

	executionsBucket := s.getExecutionsBucket(tx)
	err = executionsBucket.Put([]byte(localExecutionState.Execution.ID), data)
	if err != nil {
		return emptyState, err
	}

	// If this execution is in an active state, then we should index it so that we know
	// at a restart that it should be running. If it is not that, then we should ensure
	// that the index for live executions does not include this ID.
	var indexError error
	indexKey := []byte(localExecutionState.Execution.ID)
	bkt := s.getLiveIndexBucket(tx)

	if localExecutionState.State.IsExecuting() {
		// We can safely add an index key here even if it currently exists, so
		// we don't need to check the previous states to see if it already
		// exists.
		indexError = bkt.Put(indexKey, []byte{})
	} else {
		// Removes the index key from the bucket, and if it doesn't exist then
		// quietly returns nil as expected.
		indexError = bkt.Delete(indexKey)
	}

	if indexError != nil {
		return previousState, fmt.Errorf("failed to process live index: %s", indexError)
	}

	// Update the history of the execution
	historyErr := s.appendHistory(tx, localExecutionState, previousState, request.Comment)
	if historyErr != nil {
		return previousState, fmt.Errorf("failed to append execution history: %s", historyErr)
	}

	return previousState, nil
}

// Must be called where tx is a write transaction and adds a new history entry
// to the bucket in history/execution-id/ with a key value derived from the
// bucket sequence. Elsewhere we want to iterate through the history so to
// make sure we get it back in creation order we use a three-digit sequence
// number as bucket traversals happen in lexicographical order.
func (s *Store) appendHistory(
	tx *bolt.Tx,
	updatedExecution store.LocalExecutionState,
	previousState store.LocalExecutionStateType, comment string) error {
	historyBucket := s.getHistoryBucket(tx)
	executionHistoryBucket, err := historyBucket.CreateBucketIfNotExists([]byte(updatedExecution.Execution.ID))
	if err != nil {
		return err
	}

	historyEntry := store.LocalStateHistory{
		ExecutionID:   updatedExecution.Execution.ID,
		PreviousState: previousState,
		NewState:      updatedExecution.State,
		NewVersion:    updatedExecution.Version,
		Comment:       comment,
		Time:          updatedExecution.UpdateTime,
	}
	historyEntryData, err := s.marshaller.Marshal(historyEntry)
	if err != nil {
		return err
	}

	// History entry keys within the execution bucket will be iterated over
	// in lexographical order so we want to pad the numbers with leading 0s
	// although this adds a hard limit of 999 history items for an execution
	seqNum, err := executionHistoryBucket.NextSequence()
	if err != nil {
		return err
	}
	seq := fmt.Sprintf("%03d", seqNum)
	return executionHistoryBucket.Put([]byte(seq), historyEntryData)
}

// DeleteExecution delete the execution, removes its history and removes it
// from the job index (along with the job indexes bucket)
func (s *Store) DeleteExecution(ctx context.Context, executionID string) error {
	log.Ctx(ctx).Trace().
		Str("ExecutionID", executionID).
		Msg("boltdb.DeleteExecution")

	return s.database.Update(func(tx *bolt.Tx) (err error) {
		return s.deleteExecution(tx, executionID)
	})
}

func (s *Store) deleteExecution(tx *bolt.Tx, executionID string) error {
	localExecutionState, err := s.getExecution(tx, executionID)
	if err != nil {
		return err
	}

	// Delete single execution entry
	executionsBucket := s.getExecutionsBucket(tx)
	err = executionsBucket.Delete([]byte(executionID))
	if err != nil {
		return err
	}

	// Delete from job index
	jobIndexBucket := s.getJobIndexBucket(tx)
	jobBucket := jobIndexBucket.Bucket([]byte(localExecutionState.Execution.JobID))
	if jobBucket != nil {
		err = jobBucket.Delete([]byte(executionID))
		if err != nil {
			return err
		}
	}

	// Delete the bucket with the execution-id within the history bucket
	historyBucket := s.getHistoryBucket(tx)
	err = historyBucket.DeleteBucket([]byte(executionID))
	if err != nil {
		return err
	}

	return nil
}

// Close ensures the database is closed cleanly
func (s *Store) Close(ctx context.Context) error {
	return s.database.Close()
}

func (s *Store) GetExecutionCount(ctx context.Context, state store.LocalExecutionStateType) (uint64, error) {
	log.Ctx(ctx).Trace().
		Msg("boltdb.GetExecutionCount")

	// We have to wait here to ensure the counter has been populated,
	// so we will wait until we know it has finished
	s.starting.Wait()
	return s.stateCounter.Get(state), nil
}

func (s *Store) populateStateCounter(ctx context.Context) {
	acc := NewStateCounter()

	err := s.database.View(func(tx *bolt.Tx) (err error) {
		bucket := s.getExecutionsBucket(tx)
		cursor := bucket.Cursor()

		for k, v := cursor.First(); k != nil; k, v = cursor.Next() {
			if v == nil { // We can't do much with empty values
				continue
			}

			var entry store.LocalExecutionState
			err = s.marshaller.Unmarshal(v, &entry)
			if err != nil {
				return
			}

			acc.IncrementState(entry.State, 1)
		}

		return err
	})

	if err != nil {
		log.Ctx(ctx).Error().Err(err).Msg("failed to generate state counter for execution store")
	}

	// As more items may have been added whilst we were running, merge the
	// statecounter we have generated into the existing one for the store
	s.stateCounter.Include(acc)

	log.Ctx(ctx).Trace().Msg("finished populating state counter")
	s.starting.Done()
}

// compile-time check that we implement the interface ExecutionStore
var _ store.ExecutionStore = (*Store)(nil)
