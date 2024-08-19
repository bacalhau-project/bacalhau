package boltdb

import (
	"context"
	"encoding/binary"
	"errors"
	"fmt"
	"sync/atomic"
	"time"

	"github.com/benbjohnson/clock"
	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/rs/zerolog/log"
	"go.etcd.io/bbolt"

	"github.com/bacalhau-project/bacalhau/pkg/lib/validate"
	"github.com/bacalhau-project/bacalhau/pkg/lib/watcher"
)

// EventStore implements the EventStore interface using BoltDB as the underlying storage.
// It provides efficient storage and retrieval of events with support for caching,
// checkpointing, and garbage collection.
type EventStore struct {
	db             *bbolt.DB
	options        *eventStoreOptions
	cache          *lru.Cache[uint64, watcher.Event]
	latestEventNum atomic.Uint64
	clock          clock.Clock

	// notifyCh is a channel for notifying watchers of new events.
	// GetEvents will block on this channel when no events are immediately available,
	// or will return empty events after a long-polling timeout.
	notifyCh chan uint64
	stopGC   chan struct{}
}

// NewEventStore creates a new EventStore with the given options.
//
// It initializes the BoltDB buckets, sets up caching, and starts the garbage collection process.
// The store uses two buckets: one for events and another for checkpoints.
//
// Example usage:
//
//	db, err := bbolt.Open("events.db", 0600, nil)
//	if err != nil {
//	    // handle error
//	}
//	defer db.Close()
//
//	store, err := NewEventStore(
//	    db,
//	    WithEventsBucket("myEvents"),
//	    WithCheckpointBucket("myCheckpoints"),
//	    WithEventSerializer(NewJSONSerializer()),
//	)
//	if err != nil {
//	    // handle error
//	}
func NewEventStore(db *bbolt.DB, opts ...EventStoreOption) (*EventStore, error) {
	options := defaultEventStoreOptions()
	for _, opt := range opts {
		opt(options)
	}

	err := errors.Join(
		validate.NotNil(db, "boltDB instance cannot be nil"),
		options.validate(),
	)
	if err != nil {
		return nil, err
	}

	cache, err := lru.New[uint64, watcher.Event](options.cacheSize)
	if err != nil {
		return nil, fmt.Errorf("failed to create cache: %w", err)
	}

	store := &EventStore{
		db:       db,
		options:  options,
		cache:    cache,
		notifyCh: make(chan uint64, 100), //nolint:mnd
		clock:    options.clock,
		stopGC:   make(chan struct{}),
	}

	// Initialize BoltDB buckets
	err = db.Update(func(tx *bbolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(options.eventsBucket)
		if err != nil {
			return fmt.Errorf("failed to create events bucket: %w", err)
		}
		_, err = tx.CreateBucketIfNotExists(options.checkpointBucket)
		if err != nil {
			return fmt.Errorf("failed to create checkpoints bucket: %w", err)
		}

		b := tx.Bucket(options.eventsBucket)
		c := b.Cursor()

		k, _ := c.Last()
		if k != nil {
			var key eventKey
			if err := key.UnmarshalBinary(k); err != nil {
				return fmt.Errorf("failed to unmarshal key: %w", err)
			}
			store.latestEventNum.Store(key.SeqNum)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create buckets: %w", err)
	}

	// Start the garbage collection process
	go store.runGCLoop()
	return store, nil
}

// StoreEvent stores a new event in the EventStore.
// It wraps the storage operation in a BoltDB transaction.
func (s *EventStore) StoreEvent(ctx context.Context, operation watcher.Operation, objectType string, object interface{}) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		return s.StoreEventTx(tx, operation, objectType, object)
	})
}

// StoreEventTx stores a new event within an existing BoltDB transaction.
// It serializes the event, stores it in the database, adds it to the cache,
// and notifies watchers of the new event.
func (s *EventStore) StoreEventTx(tx *bbolt.Tx, operation watcher.Operation, objectType string, object interface{}) error {
	b := tx.Bucket(s.options.eventsBucket)
	if b == nil {
		return fmt.Errorf("events bucket not found")
	}

	id, err := b.NextSequence()
	if err != nil {
		return fmt.Errorf("failed to get next sequence: %w", err)
	}
	event := watcher.Event{
		SeqNum:     id,
		Operation:  operation,
		ObjectType: objectType,
		Object:     object,
		Timestamp:  s.clock.Now(),
	}
	eventBytes, err := s.options.serializer.Marshal(event)
	if err != nil {
		return fmt.Errorf("failed to marshal event: %w", err)
	}

	key := newEventKey(id, event.Timestamp.UnixNano())
	keyBytes, err := key.MarshalBinary()
	if err != nil {
		return fmt.Errorf("failed to marshal key: %w", err)
	}

	err = b.Put(keyBytes, eventBytes)
	if err != nil {
		return err
	}

	tx.OnCommit(func() {
		// Add to cache
		s.cache.Add(id, event)

		// Update latest event number
		s.latestEventNum.Store(id)

		// Notify watchers
		select {
		case s.notifyCh <- id:
		default:
			log.Trace().Msgf("Failed to notify watchers of new event: %d", event.SeqNum)
		}
	})

	return nil
}

// GetEvents retrieves events from the store based on the provided query parameters.
// It supports long-polling: if no events are immediately available, it waits for new events
// or until the long-polling timeout is reached.
func (s *EventStore) GetEvents(ctx context.Context, params watcher.GetEventsRequest) (*watcher.GetEventsResponse, error) {
	var result *watcher.GetEventsResponse

	for {
		err := s.db.View(func(tx *bbolt.Tx) error {
			var err error
			result, err = s.getEventsTx(tx, params)
			return err
		})

		if err != nil {
			return nil, fmt.Errorf("failed to get events: %w", err)
		}

		if len(result.Events) > 0 || result.NextEventIterator != params.EventIterator {
			return result, nil
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-s.notifyCh:
			// New events might be available, loop and check again
			// Drain any additional notifications
			for len(s.notifyCh) > 0 {
				<-s.notifyCh
			}
		case <-s.clock.After(s.options.longPollingTimeout):
			// Return empty result after timeout
			return result, nil
		}
	}
}

// getEventsTx retrieves events within a BoltDB transaction based on the provided query parameters.
// It uses the cache when possible to improve performance.
func (s *EventStore) getEventsTx(tx *bbolt.Tx, params watcher.GetEventsRequest) (*watcher.GetEventsResponse, error) {
	b := tx.Bucket(s.options.eventsBucket)
	c := b.Cursor()

	iterator, err := s.resolveIteratorType(tx, params)
	if err != nil {
		return nil, err
	}

	// results to return
	events := make([]watcher.Event, 0)
	nextIterator := iterator

	startKey := make([]byte, seqNumBytes)
	binary.BigEndian.PutUint64(startKey, iterator.SequenceNumber)

	for k, v := c.Seek(startKey); k != nil && (params.Limit == 0 || len(events) < params.Limit); k, v = c.Next() {
		var key eventKey
		if err = key.UnmarshalBinary(k); err != nil {
			return nil, fmt.Errorf("failed to unmarshal key: %w", err)
		}
		if key.SeqNum < iterator.SequenceNumber {
			continue // Skip this event as it's before the start key
		} else if key.SeqNum == iterator.SequenceNumber && iterator.Type == watcher.EventIteratorAfterSequenceNumber {
			continue // Skip this event as it's the same as the start key and we're looking for events after it
		}

		// Update the next iterator to after the current key, even if we filter it out
		nextIterator.Type = watcher.EventIteratorAfterSequenceNumber
		nextIterator.SequenceNumber = key.SeqNum

		event, found := s.cache.Get(key.SeqNum)
		if !found {
			if err = s.options.serializer.Unmarshal(v, &event); err != nil {
				return nil, fmt.Errorf("failed to unmarshal event: %w", err)
			}
		}

		if filterEvent(event, params.Filter) {
			events = append(events, event)
		}
	}

	return &watcher.GetEventsResponse{
		Events:            events,
		NextEventIterator: nextIterator,
	}, nil
}

// resolveIteratorType resolves the event iterator type based on the provided query parameters.
// It returns a new event iterator if the type is set to EventIteratorTrimHorizon or EventIteratorLatest.
func (s *EventStore) resolveIteratorType(tx *bbolt.Tx, params watcher.GetEventsRequest) (watcher.EventIterator, error) {
	switch params.EventIterator.Type {
	case watcher.EventIteratorTrimHorizon:
		return watcher.AfterSequenceNumberIterator(0), nil
	case watcher.EventIteratorLatest:
		seqNum, err := s.GetLatestEventNum(context.Background())
		if err != nil {
			return watcher.EventIterator{}, fmt.Errorf("failed to get latest event num: %w", err)
		}
		return watcher.AfterSequenceNumberIterator(seqNum), nil
	default:
		return params.EventIterator, nil
	}
}

// GetLatestEventNum retrieves the sequence number of the latest event in the store.
func (s *EventStore) GetLatestEventNum(ctx context.Context) (uint64, error) {
	return s.latestEventNum.Load(), nil
}

// StoreCheckpoint stores a checkpoint for a specific watcher.
// Checkpoints are used to track which events have been processed by each watcher.
func (s *EventStore) StoreCheckpoint(ctx context.Context, watcherID string, eventSeqNum uint64) error {
	return s.db.Update(func(tx *bbolt.Tx) error {
		return s.storeCheckpointTx(tx, watcherID, eventSeqNum)
	})
}

// storeCheckpointTx stores a checkpoint within an existing BoltDB transaction.
func (s *EventStore) storeCheckpointTx(tx *bbolt.Tx, watcherID string, eventSeqNum uint64) error {
	b := tx.Bucket(s.options.checkpointBucket)
	value := make([]byte, seqNumBytes)
	binary.BigEndian.PutUint64(value, eventSeqNum)
	return b.Put([]byte(watcherID), value)
}

// GetCheckpoint retrieves the checkpoint for a specific watcher.
// If the checkpoint is not found, it returns a CheckpointError.
func (s *EventStore) GetCheckpoint(ctx context.Context, watcherID string) (uint64, error) {
	var checkpoint uint64

	err := s.db.View(func(tx *bbolt.Tx) error {
		return s.getCheckpointTx(tx, watcherID, &checkpoint)
	})

	if err != nil {
		var checkpointErr *watcher.CheckpointError
		if errors.As(err, &checkpointErr) {
			return 0, err
		}
		return 0, watcher.NewCheckpointError(watcherID, err)
	}

	return checkpoint, nil
}

// getCheckpointTx retrieves a checkpoint within an existing BoltDB transaction.
func (s *EventStore) getCheckpointTx(tx *bbolt.Tx, watcherID string, checkpoint *uint64) error {
	b := tx.Bucket(s.options.checkpointBucket)
	value := b.Get([]byte(watcherID))
	if value == nil {
		return watcher.NewCheckpointError(watcherID, watcher.ErrCheckpointNotFound)
	}
	*checkpoint = binary.BigEndian.Uint64(value)
	return nil
}

// filterEvent checks if an event matches the given filter criteria.
func filterEvent(event watcher.Event, filter watcher.EventFilter) bool {
	if len(filter.ObjectTypes) > 0 && !contains(filter.ObjectTypes, event.ObjectType) {
		return false
	}

	if len(filter.Operations) > 0 && !containsOperation(filter.Operations, event.Operation) {
		return false
	}

	return true
}

// runGCLoop runs the garbage collection process at regular intervals.
func (s *EventStore) runGCLoop() {
	if s.options.gcAgeThreshold == 0 {
		return // GC is disabled if threshold is not set
	}

	ticker := s.clock.Ticker(s.options.gcCadence)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			if err := s.runGC(); err != nil {
				log.Error().Err(err).Msg("Error during garbage collection. Continuing...")
			}
		case <-s.stopGC:
			return
		}
	}
}

// runGC performs garbage collection by pruning old events that are beyond all watchers' checkpoints
// and older than the configured event age threshold.
func (s *EventStore) runGC() error {
	if s.options.gcAgeThreshold == 0 {
		return nil // GC is disabled if threshold is not set
	}

	var minCheckpoint uint64
	var err error

	// Find the minimum checkpoint across all watchers
	err = s.db.View(func(tx *bbolt.Tx) error {
		b := tx.Bucket(s.options.checkpointBucket)
		c := b.Cursor()

		for k, v := c.First(); k != nil; k, v = c.Next() {
			checkpoint := binary.BigEndian.Uint64(v)
			if minCheckpoint == 0 || checkpoint < minCheckpoint {
				minCheckpoint = checkpoint
			}
		}
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to find minimum checkpoint: %w", err)
	}

	// Prune events
	return s.db.Update(func(tx *bbolt.Tx) error {
		b := tx.Bucket(s.options.eventsBucket)
		c := b.Cursor()

		var processed int
		var deleted int
		var latestSeqNum uint64

		defer func() {
			log.Debug().Msgf("GC deleted %d events upto %d", deleted, latestSeqNum)
		}()

		start := s.clock.Now()
		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			var key eventKey
			if err = key.UnmarshalBinary(k); err != nil {
				return fmt.Errorf("failed to unmarshal event key: %w", err)
			}

			if key.SeqNum > minCheckpoint {
				break // We've reached events that are still needed
			}

			if s.clock.Since(time.Unix(0, key.Timestamp)) > s.options.gcAgeThreshold {
				if err = c.Delete(); err != nil {
					return fmt.Errorf("failed to delete old event: %w", err)
				}
				deleted++
				latestSeqNum = key.SeqNum
			}

			processed++
			if processed >= s.options.gcMaxRecordsPerRun || s.clock.Since(start) >= s.options.gcMaxDuration {
				break // Stop GC for this run to avoid holding the lock too long
			}
		}

		return nil
	})
}

// Close stops the garbage collection process and purges the cache.
func (s *EventStore) Close(ctx context.Context) error {
	close(s.stopGC)
	s.cache.Purge()
	return nil
}

// compile-time check for interface conformance
var _ watcher.EventStore = &EventStore{}
