//go:build unit || !integration

package watcher_test

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/bacalhau-project/bacalhau/pkg/lib/watcher"
	"github.com/bacalhau-project/bacalhau/pkg/lib/watcher/boltdb"
	watchertest "github.com/bacalhau-project/bacalhau/pkg/lib/watcher/test"
)

type WatcherTestSuite struct {
	suite.Suite
	ctrl        *gomock.Controller
	mockStore   *watchertest.EventStoreWrapper
	mockHandler *watcher.MockEventHandler
	registry    watcher.Registry
}

func (s *WatcherTestSuite) SetupTest() {
	boltdbEventStore, err := boltdb.NewEventStore(watchertest.CreateBoltDB(s.T()),
		boltdb.WithLongPollingTimeout(1*time.Second),
		boltdb.WithEventSerializer(watchertest.CreateSerializer(s.T())),
	)
	s.Require().NoError(err)

	s.ctrl = gomock.NewController(s.T())
	s.mockStore = watchertest.NewEventStoreWrapper(boltdbEventStore)
	s.mockHandler = watcher.NewMockEventHandler(s.ctrl)
	s.registry = watcher.NewRegistry(s.mockStore)
}

func (s *WatcherTestSuite) TearDownTest() {
	s.ctrl.Finish()
	s.registry.Stop(context.Background())
}

func (s *WatcherTestSuite) TestCreateWatcher() {
	ctx := context.Background()
	w, err := s.registry.Watch(ctx, "test-watcher", s.mockHandler)
	s.Require().NoError(err)
	s.Require().NotNil(w)
	s.Equal("test-watcher", w.ID())
	s.Eventually(func() bool { return w.Stats().State == watcher.StateRunning }, 200*time.Millisecond, 10*time.Millisecond)

	// verify stats
	stats := w.Stats()
	s.Equal("test-watcher", stats.ID)
	s.Equal(watcher.StateRunning, stats.State)
	s.Equal(uint64(0), stats.LastProcessedSeqNum)
	s.Equal(time.Time{}, stats.LastProcessedEventTime)

	// Stop the watcher
	w.Stop(ctx)
	s.Eventually(func() bool { return w.Stats().State == watcher.StateStopped }, 200*time.Millisecond, 10*time.Millisecond)
}

func (s *WatcherTestSuite) TestWatcherProcessEvents() {
	ctx := context.Background()
	events := []watcher.Event{
		{Operation: watcher.OperationCreate, ObjectType: "TestObject", Object: "test1"},
		{Operation: watcher.OperationUpdate, ObjectType: "TestObject", Object: "test2"},
	}

	for _, event := range events {
		err := s.mockStore.StoreEvent(ctx, event.Operation, event.ObjectType, event.Object)
		s.Require().NoError(err)
	}

	gomock.InOrder(
		s.mockHandler.EXPECT().HandleEvent(gomock.Any(), watchertest.EventWithSeqNum(1)).Return(nil).Times(1),
		s.mockHandler.EXPECT().HandleEvent(gomock.Any(), watchertest.EventWithSeqNum(2)).Return(nil).Times(1),
	)

	w, err := s.registry.Watch(ctx, "test-watcher", s.mockHandler)
	s.Require().NoError(err)

	s.waitAndStop(ctx, w, 2)
}

func (s *WatcherTestSuite) TestWithStartSeqNum() {
	ctx := context.Background()
	events := []watcher.Event{
		{Operation: watcher.OperationCreate, ObjectType: "TestObject", Object: "test1"},
		{Operation: watcher.OperationUpdate, ObjectType: "TestObject", Object: "test2"},
		{Operation: watcher.OperationDelete, ObjectType: "TestObject", Object: "test3"},
	}

	for _, event := range events {
		err := s.mockStore.StoreEvent(ctx, event.Operation, event.ObjectType, event.Object)
		s.Require().NoError(err)
	}

	gomock.InOrder(
		s.mockHandler.EXPECT().HandleEvent(gomock.Any(), watchertest.EventWithSeqNum(2)).Return(nil).Times(1),
		s.mockHandler.EXPECT().HandleEvent(gomock.Any(), watchertest.EventWithSeqNum(3)).Return(nil).Times(1),
	)

	w, err := s.registry.Watch(ctx, "test-watcher", s.mockHandler,
		watcher.WithInitialEventIterator(watcher.AfterSequenceNumberIterator(1)))
	s.Require().NoError(err)

	s.waitAndStop(ctx, w, 3)
}

func (s *WatcherTestSuite) TestWithFilter() {
	ctx := context.Background()
	filter := watcher.EventFilter{
		ObjectTypes: []string{"TestObject"},
		Operations:  []watcher.Operation{watcher.OperationCreate, watcher.OperationUpdate},
	}
	events := []watcher.Event{
		{SeqNum: 1, Operation: watcher.OperationCreate, ObjectType: "TestObject", Object: "test1"},
		{SeqNum: 2, Operation: watcher.OperationUpdate, ObjectType: "TestObject", Object: "test2"},
		{SeqNum: 3, Operation: watcher.OperationDelete, ObjectType: "TestObject", Object: "test3"},
		{SeqNum: 4, Operation: watcher.OperationCreate, ObjectType: "TestObject", Object: "test4"},
		{SeqNum: 5, Operation: watcher.OperationCreate, ObjectType: "OtherObject", Object: "test5"},
	}

	for _, event := range events {
		err := s.mockStore.StoreEvent(ctx, event.Operation, event.ObjectType, event.Object)
		s.Require().NoError(err)
	}

	gomock.InOrder(
		s.mockHandler.EXPECT().HandleEvent(gomock.Any(), watchertest.EventWithSeqNum(1)).Return(nil).Times(1),
		s.mockHandler.EXPECT().HandleEvent(gomock.Any(), watchertest.EventWithSeqNum(2)).Return(nil).Times(1),
		s.mockHandler.EXPECT().HandleEvent(gomock.Any(), watchertest.EventWithSeqNum(4)).Return(nil).Times(1),
	)

	w, err := s.registry.Watch(ctx, "test-watcher", s.mockHandler, watcher.WithFilter(filter))
	s.Require().NoError(err)

	s.waitAndStop(ctx, w, 5)
	s.Equal(uint64(4), w.Stats().LastProcessedSeqNum) // last event not processed
}

func (s *WatcherTestSuite) TestCheckpoint() {
	ctx := context.Background()
	events := []watcher.Event{
		{SeqNum: 1, Operation: watcher.OperationCreate, ObjectType: "TestObject", Object: "test1"},
		{SeqNum: 2, Operation: watcher.OperationUpdate, ObjectType: "TestObject", Object: "test2"},
	}

	for _, event := range events {
		err := s.mockStore.StoreEvent(ctx, event.Operation, event.ObjectType, event.Object)
		s.Require().NoError(err)
	}

	gomock.InOrder(
		s.mockHandler.EXPECT().HandleEvent(gomock.Any(), watchertest.EventWithSeqNum(1)).Return(nil).Times(1),
		s.mockHandler.EXPECT().HandleEvent(gomock.Any(), watchertest.EventWithSeqNum(2)).Return(nil).Times(1),
	)
	w, err := s.registry.Watch(ctx, "test-watcher", s.mockHandler)
	s.Require().NoError(err)

	s.wait(ctx, w, 2)

	// Manually checkpoint
	err = w.Checkpoint(ctx, 1)
	s.Require().NoError(err)

	w.Stop(ctx)

	// Verify the checkpoint
	checkpoint, err := s.mockStore.GetCheckpoint(ctx, w.ID())
	s.Require().NoError(err)
	s.Equal(uint64(1), checkpoint)

	// Start a new watcher and verify that it starts from the checkpoint
	newHandler := watcher.NewMockEventHandler(s.ctrl)
	newHandler.EXPECT().HandleEvent(gomock.Any(), watchertest.EventWithSeqNum(2)).Return(nil).Times(1)

	w, err = s.registry.Watch(ctx, "test-watcher", newHandler)
	s.Require().NoError(err)

	s.waitAndStop(ctx, w, 2)
}

func (s *WatcherTestSuite) TestSeekToOffset() {
	ctx := context.Background()
	events := []watcher.Event{
		{SeqNum: 1, Operation: watcher.OperationCreate, ObjectType: "TestObject", Object: "test1"},
		{SeqNum: 2, Operation: watcher.OperationUpdate, ObjectType: "TestObject", Object: "test2"},
		{SeqNum: 3, Operation: watcher.OperationDelete, ObjectType: "TestObject", Object: "test3"},
	}

	for _, event := range events {
		err := s.mockStore.StoreEvent(ctx, event.Operation, event.ObjectType, event.Object)
		s.Require().NoError(err)
	}

	gomock.InOrder(
		s.mockHandler.EXPECT().HandleEvent(gomock.Any(), watchertest.EventWithSeqNum(1)).Return(nil).Times(1),
		s.mockHandler.EXPECT().HandleEvent(gomock.Any(), watchertest.EventWithSeqNum(2)).Return(nil).Times(1),

		// last event is processed twice after seek
		s.mockHandler.EXPECT().HandleEvent(gomock.Any(), watchertest.EventWithSeqNum(3)).Return(nil).Times(2),
	)
	w, err := s.registry.Watch(ctx, "test-watcher", s.mockHandler)
	s.Require().NoError(err)
	s.wait(ctx, w, 3)

	// Seek to offset 2
	err = w.SeekToOffset(ctx, 2)
	s.Require().NoError(err)

	// Verify the checkpoint
	checkpoint, err := s.mockStore.GetCheckpoint(ctx, w.ID())
	s.Require().NoError(err)
	s.Equal(uint64(2), checkpoint)

	s.waitAndStop(ctx, w, 3)
}

func (s *WatcherTestSuite) TestCheckpointAndStartSeqNum() {
	ctx := context.Background()

	testCases := []struct {
		name           string
		checkpoint     uint64
		startSeqNum    uint64
		expectedEvents []uint64
	}{
		{
			name:           "No checkpoint, no startSeqNum",
			checkpoint:     0,
			startSeqNum:    0,
			expectedEvents: []uint64{1, 2, 3, 4, 5},
		},
		{
			name:           "Checkpoint, no startSeqNum",
			checkpoint:     2,
			startSeqNum:    0,
			expectedEvents: []uint64{3, 4, 5},
		},
		{
			name:           "No checkpoint, with startSeqNum",
			checkpoint:     0,
			startSeqNum:    3,
			expectedEvents: []uint64{4, 5},
		},
		{
			name:           "Checkpoint lower than startSeqNum",
			checkpoint:     2,
			startSeqNum:    4,
			expectedEvents: []uint64{3, 4, 5},
		},
		{
			name:           "Checkpoint higher than startSeqNum",
			checkpoint:     4,
			startSeqNum:    2,
			expectedEvents: []uint64{5},
		},
		{
			name:           "Checkpoint equal to startSeqNum",
			checkpoint:     3,
			startSeqNum:    3,
			expectedEvents: []uint64{4, 5},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// start fresh with each run
			s.TearDownTest()
			s.SetupTest()

			// Create some initial events
			events := []watcher.Event{
				{SeqNum: 1, Operation: watcher.OperationCreate, ObjectType: "TestObject", Object: "test1"},
				{SeqNum: 2, Operation: watcher.OperationUpdate, ObjectType: "TestObject", Object: "test2"},
				{SeqNum: 3, Operation: watcher.OperationDelete, ObjectType: "TestObject", Object: "test3"},
				{SeqNum: 4, Operation: watcher.OperationCreate, ObjectType: "TestObject", Object: "test4"},
				{SeqNum: 5, Operation: watcher.OperationUpdate, ObjectType: "TestObject", Object: "test5"},
			}

			for _, event := range events {
				err := s.mockStore.StoreEvent(ctx, event.Operation, event.ObjectType, event.Object)
				s.Require().NoError(err)
			}

			// Set up the checkpoint if provided
			if tc.checkpoint != 0 {
				err := s.mockStore.StoreCheckpoint(ctx, "test-watcher", tc.checkpoint)
				s.Require().NoError(err)
			}

			// Set up expectations
			for _, expectedSeqNum := range tc.expectedEvents {
				s.mockHandler.EXPECT().HandleEvent(gomock.Any(), watchertest.EventWithSeqNum(expectedSeqNum)).Return(nil).Times(1)
			}

			// Start the watcher
			w, err := s.registry.Watch(ctx, "test-watcher", s.mockHandler,
				watcher.WithInitialEventIterator(watcher.AfterSequenceNumberIterator(tc.startSeqNum)))
			s.Require().NoError(err)

			// Wait for processing and stop
			s.waitAndStop(ctx, w, uint64(5))
		})
	}
}

func (s *WatcherTestSuite) TestHandleEventErrorWithBlockStrategy() {
	ctx := context.Background()
	events := []watcher.Event{
		{SeqNum: 1, Operation: watcher.OperationCreate, ObjectType: "TestObject", Object: "test1"},
		{SeqNum: 2, Operation: watcher.OperationUpdate, ObjectType: "TestObject", Object: "test2"},
	}

	for _, event := range events {
		err := s.mockStore.StoreEvent(ctx, event.Operation, event.ObjectType, event.Object)
		s.Require().NoError(err)
	}

	failCount := 0
	maxFails := 100

	gomock.InOrder(
		s.mockHandler.EXPECT().HandleEvent(gomock.Any(), watchertest.EventWithSeqNum(1)).
			DoAndReturn(func(_ context.Context, _ watcher.Event) error {
				failCount++
				if failCount <= maxFails {
					return errors.New("handling error")
				}
				return nil
			}).Times(maxFails+1),
		s.mockHandler.EXPECT().HandleEvent(gomock.Any(), watchertest.EventWithSeqNum(2)).Return(nil).Times(1),
	)

	w, err := s.registry.Watch(ctx, "test-watcher", s.mockHandler,
		watcher.WithMaxRetries(3), // will be ignored with block strategy
		watcher.WithInitialBackoff(1*time.Nanosecond),
		watcher.WithMaxBackoff(1*time.Nanosecond),
		watcher.WithRetryStrategy(watcher.RetryStrategyBlock))
	s.Require().NoError(err)

	s.waitAndStop(ctx, w, 2)
	s.Equal(uint64(2), w.Stats().LastProcessedSeqNum)
	s.Equal(maxFails+1, failCount) // Verify that it failed 100 times before succeeding
}

func (s *WatcherTestSuite) TestDifferentIteratorTypes() {
	ctx := context.Background()
	events := []watcher.Event{
		{Operation: watcher.OperationCreate, ObjectType: "TestObject", Object: "test1"},
		{Operation: watcher.OperationUpdate, ObjectType: "TestObject", Object: "test2"},
		{Operation: watcher.OperationDelete, ObjectType: "TestObject", Object: "test3"},
		{Operation: watcher.OperationCreate, ObjectType: "TestObject", Object: "test4"},
	}

	for _, event := range events {
		err := s.mockStore.StoreEvent(ctx, event.Operation, event.ObjectType, event.Object)
		s.Require().NoError(err)
	}

	testCases := []struct {
		name           string
		iterator       watcher.EventIterator
		expectedEvents []uint64
	}{
		{
			name:           "TrimHorizon",
			iterator:       watcher.TrimHorizonIterator(),
			expectedEvents: []uint64{1, 2, 3, 4},
		},
		{
			name:           "Latest",
			iterator:       watcher.LatestIterator(),
			expectedEvents: []uint64{},
		},
		{
			name:           "AtSequenceNumber(2)",
			iterator:       watcher.AtSequenceNumberIterator(2),
			expectedEvents: []uint64{2, 3, 4},
		},
		{
			name:           "AfterSequenceNumber(2)",
			iterator:       watcher.AfterSequenceNumberIterator(2),
			expectedEvents: []uint64{3, 4},
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.mockHandler = watcher.NewMockEventHandler(s.ctrl)
			for _, seqNum := range tc.expectedEvents {
				s.mockHandler.EXPECT().HandleEvent(gomock.Any(), watchertest.EventWithSeqNum(seqNum)).Return(nil).Times(1)
			}

			w, err := s.registry.Watch(ctx, "test-watcher", s.mockHandler,
				watcher.WithInitialEventIterator(tc.iterator))
			s.Require().NoError(err)

			s.waitAndStop(ctx, w, 4)
		})
	}
}

func (s *WatcherTestSuite) TestEmptyEventStoreWithDifferentIterators() {
	ctx := context.Background()

	testCases := []struct {
		name             string
		iterator         watcher.EventIterator
		expectedIterator watcher.EventIterator
	}{
		{
			name:             "TrimHorizon",
			iterator:         watcher.TrimHorizonIterator(),
			expectedIterator: watcher.AfterSequenceNumberIterator(0),
		},
		{
			name:             "Latest",
			iterator:         watcher.LatestIterator(),
			expectedIterator: watcher.AfterSequenceNumberIterator(0),
		},
		{
			name:             "AtSequenceNumber(1)",
			iterator:         watcher.AtSequenceNumberIterator(1),
			expectedIterator: watcher.AtSequenceNumberIterator(1),
		},
		{
			name:             "AfterSequenceNumber(0)",
			iterator:         watcher.AfterSequenceNumberIterator(0),
			expectedIterator: watcher.AfterSequenceNumberIterator(0),
		},
		{
			name:             "AfterSequenceNumber(1)",
			iterator:         watcher.AfterSequenceNumberIterator(1),
			expectedIterator: watcher.AfterSequenceNumberIterator(1),
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.mockHandler = watcher.NewMockEventHandler(s.ctrl)

			// Test with empty store
			w, err := s.registry.Watch(ctx, "test-watcher", s.mockHandler,
				watcher.WithInitialEventIterator(tc.iterator))
			s.Require().NoError(err)
			s.waitAndStop(ctx, w, tc.expectedIterator.SequenceNumber)

			// Check if the next event iterator matches the expected iterator
			s.Equal(tc.expectedIterator, w.Stats().NextEventIterator)
		})
	}
}

func (s *WatcherTestSuite) TestHandleEventErrorWithSkipStrategy() {
	ctx := context.Background()
	events := []watcher.Event{
		{SeqNum: 1, Operation: watcher.OperationCreate, ObjectType: "TestObject", Object: "test1"},
		{SeqNum: 2, Operation: watcher.OperationUpdate, ObjectType: "TestObject", Object: "test2"},
	}

	for _, event := range events {
		err := s.mockStore.StoreEvent(ctx, event.Operation, event.ObjectType, event.Object)
		s.Require().NoError(err)
	}

	gomock.InOrder(
		s.mockHandler.EXPECT().HandleEvent(gomock.Any(), watchertest.EventWithSeqNum(1)).Return(errors.New("handling error")).Times(3),
		s.mockHandler.EXPECT().HandleEvent(gomock.Any(), watchertest.EventWithSeqNum(2)).Return(nil).Times(1),
	)

	w, err := s.registry.Watch(ctx, "test-watcher", s.mockHandler,
		watcher.WithMaxRetries(3),
		watcher.WithInitialBackoff(1*time.Nanosecond),
		watcher.WithRetryStrategy(watcher.RetryStrategySkip))
	s.Require().NoError(err)

	s.waitAndStop(ctx, w, 2)
	s.Equal(uint64(2), w.Stats().LastProcessedSeqNum)
}

func (s *WatcherTestSuite) TestBatchOptions() {
	ctx := context.Background()

	testCases := []struct {
		name          string
		batchSize     int
		eventCount    int
		expectedCalls int
	}{
		{name: "tiny-batch", batchSize: 1, eventCount: 5, expectedCalls: 5},
		{name: "small-batch", batchSize: 2, eventCount: 5, expectedCalls: 3},
		{name: "large-batch", batchSize: 10, eventCount: 5, expectedCalls: 1},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// start fresh with each run
			s.TearDownTest()
			s.SetupTest()

			for i := 0; i < tc.eventCount; i++ {
				err := s.mockStore.StoreEvent(ctx, watcher.OperationCreate, "TestObject", fmt.Sprintf("test%d", i+1))
				s.Require().NoError(err)
			}

			getEventsCallCount := 0
			s.mockStore.WithGetEventsInterceptor(func() error {
				getEventsCallCount++
				return nil
			})

			w, err := s.registry.Watch(ctx, "test-watcher", s.mockHandler,
				watcher.WithBatchSize(tc.batchSize),
			)
			s.Require().NoError(err)

			s.mockHandler.EXPECT().HandleEvent(gomock.Any(), gomock.Any()).Return(nil).Times(tc.eventCount)

			s.waitAndStop(ctx, w, uint64(tc.eventCount))
			s.Equal(tc.expectedCalls+1, getEventsCallCount) // one extra call longpolling for new events
		})
	}
}

func (s *WatcherTestSuite) TestFilterEdgeCases() {
	ctx := context.Background()

	testCases := []struct {
		name           string
		filter         watcher.EventFilter
		events         []watcher.Event
		expectedEvents int
	}{
		{
			name:   "Empty filter",
			filter: watcher.EventFilter{},
			events: []watcher.Event{
				{Operation: watcher.OperationCreate, ObjectType: "TestObject", Object: "test1"},
				{Operation: watcher.OperationUpdate, ObjectType: "OtherObject", Object: "test2"},
			},
			expectedEvents: 2,
		},
		{
			name: "Multiple criteria",
			filter: watcher.EventFilter{
				ObjectTypes: []string{"TestObject"},
				Operations:  []watcher.Operation{watcher.OperationCreate},
			},
			events: []watcher.Event{
				{Operation: watcher.OperationCreate, ObjectType: "TestObject", Object: "test1"},
				{Operation: watcher.OperationUpdate, ObjectType: "TestObject", Object: "test2"},
				{Operation: watcher.OperationCreate, ObjectType: "OtherObject", Object: "test3"},
			},
			expectedEvents: 1,
		},
		{
			name:   "No matching events",
			filter: watcher.EventFilter{ObjectTypes: []string{"NonExistentType"}},
			events: []watcher.Event{
				{Operation: watcher.OperationCreate, ObjectType: "TestObject", Object: "test1"},
				{Operation: watcher.OperationUpdate, ObjectType: "OtherObject", Object: "test2"},
			},
			expectedEvents: 0,
		},
	}

	for _, tc := range testCases {
		// start fresh with each run
		s.TearDownTest()
		s.SetupTest()

		s.Run(tc.name, func() {
			for _, event := range tc.events {
				err := s.mockStore.StoreEvent(ctx, event.Operation, event.ObjectType, event.Object)
				s.Require().NoError(err)
			}

			w, err := s.registry.Watch(ctx, "test-watcher", s.mockHandler,
				watcher.WithFilter(tc.filter))
			s.Require().NoError(err)

			s.mockHandler.EXPECT().HandleEvent(gomock.Any(), gomock.Any()).
				Return(nil).Times(tc.expectedEvents)

			s.waitAndStop(ctx, w, uint64(len(tc.events)))
			s.Equal(uint64(tc.expectedEvents), w.Stats().LastProcessedSeqNum)
		})
	}
}

func (s *WatcherTestSuite) TestRestartBehavior() {
	ctx := context.Background()
	events := []watcher.Event{
		{Operation: watcher.OperationCreate, ObjectType: "TestObject", Object: "test1"},
		{Operation: watcher.OperationUpdate, ObjectType: "TestObject", Object: "test2"},
	}

	for _, event := range events {
		err := s.mockStore.StoreEvent(ctx, event.Operation, event.ObjectType, event.Object)
		s.Require().NoError(err)
	}

	for i := 0; i < 3; i++ {
		handler := watcher.NewMockEventHandler(s.ctrl)
		gomock.InOrder(
			handler.EXPECT().HandleEvent(gomock.Any(), watchertest.EventWithSeqNum(1)).Return(nil).Times(1),
			handler.EXPECT().HandleEvent(gomock.Any(), watchertest.EventWithSeqNum(2)).Return(nil).Times(1),
		)

		w, err := s.registry.Watch(ctx, "test-watcher", handler)
		s.Require().NoError(err)
		s.waitAndStop(ctx, w, 2)
	}
}

func (s *WatcherTestSuite) TestListenAndStoreConcurrently() {
	ctx := context.Background()

	w, err := s.registry.Watch(ctx, "test-watcher", s.mockHandler,
		watcher.WithBatchSize(7))
	s.Require().NoError(err)
	s.wait(ctx, w, 0)

	eventCount := 100

	// Set up expectations
	expectations := make([]any, eventCount)
	for i := 0; i < eventCount; i++ {
		expectations[i] = s.mockHandler.EXPECT().HandleEvent(gomock.Any(), watchertest.EventWithSeqNum(uint64(i+1))).Return(nil).Times(1)
	}

	// in a separate go routine, loop and store evens
	done := make(chan struct{})
	go func() {
		for i := 0; i < eventCount; i++ {
			err = s.mockStore.StoreEvent(ctx, watcher.OperationCreate, "TestObject", fmt.Sprintf("test%d", i))
			s.Require().NoError(err)
		}
		close(done)
	}()

	<-done
	gomock.InOrder(expectations...)

	s.waitAndStop(ctx, w, uint64(eventCount))
}

func (s *WatcherTestSuite) TestEventStoreConsistency() {
	ctx := context.Background()

	w, err := s.registry.Watch(ctx, "test-watcher", s.mockHandler,
		watcher.WithBatchSize(2))
	s.Require().NoError(err)

	// Set up expectations
	gomock.InOrder(
		s.mockHandler.EXPECT().HandleEvent(gomock.Any(), watchertest.EventWithSeqNum(1)).DoAndReturn(
			func(ctx context.Context, event watcher.Event) error {
				// Store more events while processing
				s.mockStore.StoreEvent(ctx, watcher.OperationCreate, "TestObject", "test")
				s.mockStore.StoreEvent(ctx, watcher.OperationCreate, "TestObject", "test")
				return nil
			}).Times(1),
		s.mockHandler.EXPECT().HandleEvent(gomock.Any(), watchertest.EventWithSeqNum(2)).Return(nil).Times(1),
		s.mockHandler.EXPECT().HandleEvent(gomock.Any(), watchertest.EventWithSeqNum(3)).Return(nil).Times(1),
		s.mockHandler.EXPECT().HandleEvent(gomock.Any(), watchertest.EventWithSeqNum(4)).Return(nil).Times(1),
	)

	// Store initial events
	s.mockStore.StoreEvent(ctx, watcher.OperationCreate, "TestObject", "test")
	s.mockStore.StoreEvent(ctx, watcher.OperationCreate, "TestObject", "test")

	s.waitAndStop(ctx, w, 4)
}

func (s *WatcherTestSuite) wait(ctx context.Context, w watcher.Watcher, continuationSeqNum uint64) {
	// wait for the watcher to consume the events
	s.Eventually(func() bool { return w.Stats().NextEventIterator.SequenceNumber == continuationSeqNum }, 1*time.Second, 10*time.Millisecond)
	s.Equal(watcher.StateRunning, w.Stats().State)
}

func (s *WatcherTestSuite) waitAndStop(ctx context.Context, w watcher.Watcher, continuationSeqNum uint64) {
	s.wait(ctx, w, continuationSeqNum)
	w.Stop(ctx)
	s.Eventually(func() bool { return w.Stats().State == watcher.StateStopped }, 1*time.Second, 10*time.Millisecond)
}

func TestWatcherSuite(t *testing.T) {
	suite.Run(t, new(WatcherTestSuite))
}
