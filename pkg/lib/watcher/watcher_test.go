//go:build unit || !integration

package watcher_test

import (
	"context"
	"errors"
	"fmt"
	"reflect"
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
	serializer := watchertest.CreateSerializer(s.T())
	s.Require().NoError(serializer.RegisterType("StringObject", reflect.TypeOf("")))
	s.Require().NoError(serializer.RegisterType("OtherStringObject", reflect.TypeOf("")))

	boltdbEventStore, err := boltdb.NewEventStore(watchertest.CreateBoltDB(s.T()),
		boltdb.WithLongPollingTimeout(1*time.Second),
		boltdb.WithEventSerializer(serializer),
		boltdb.WithCacheSize(2), // smaller cache size to trigger eviction and unmarshalling of boltdb events
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
func (s *WatcherTestSuite) TestDetermineStartingIterator() {
	ctx := context.Background()

	testCases := []struct {
		name             string
		setupCheckpoint  *uint64 // pointer to handle nil case
		initialIter      watcher.EventIterator
		setupLatestEvent *uint64 // what should be the latest event in store before test
		expectedIter     watcher.EventIterator
		expectedError    bool
		checkpointErr    error
		latestErr        error
	}{
		{
			name:         "No checkpoint, non-latest iterator",
			initialIter:  watcher.AfterSequenceNumberIterator(5),
			expectedIter: watcher.AfterSequenceNumberIterator(5),
		},
		{
			name:            "With checkpoint, non-latest iterator",
			setupCheckpoint: ptr(uint64(10)),
			initialIter:     watcher.AfterSequenceNumberIterator(5),
			expectedIter:    watcher.AfterSequenceNumberIterator(10),
		},
		{
			name:             "No checkpoint, latest iterator",
			initialIter:      watcher.LatestIterator(),
			setupLatestEvent: ptr(uint64(15)), // Store event up to seq 15
			expectedIter:     watcher.AfterSequenceNumberIterator(15),
		},
		{
			name:             "With checkpoint, latest iterator",
			setupCheckpoint:  ptr(uint64(10)),
			initialIter:      watcher.LatestIterator(),
			setupLatestEvent: ptr(uint64(15)),
			expectedIter:     watcher.AfterSequenceNumberIterator(10),
		},
		{
			name:          "Checkpoint error",
			initialIter:   watcher.AfterSequenceNumberIterator(5),
			checkpointErr: errors.New("db error"),
			expectedError: true,
		},
		{
			name:          "Latest error",
			initialIter:   watcher.LatestIterator(),
			latestErr:     errors.New("db error"),
			expectedError: true,
		},
		{
			name:         "Empty store, latest iterator",
			initialIter:  watcher.LatestIterator(),
			expectedIter: watcher.AfterSequenceNumberIterator(0),
		},
		{
			name:            "TrimHorizon with checkpoint",
			setupCheckpoint: ptr(uint64(10)),
			initialIter:     watcher.TrimHorizonIterator(),
			expectedIter:    watcher.AfterSequenceNumberIterator(10),
		},
		{
			name:         "TrimHorizon without checkpoint",
			initialIter:  watcher.TrimHorizonIterator(),
			expectedIter: watcher.TrimHorizonIterator(),
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			// start fresh with each run
			s.TearDownTest()
			s.SetupTest()

			// Setup initial state if needed
			if tc.setupLatestEvent != nil {
				// Store events up to the desired sequence number
				for i := uint64(1); i <= *tc.setupLatestEvent; i++ {
					err := s.mockStore.StoreEvent(ctx, watcher.StoreEventRequest{
						Operation:  watcher.OperationCreate,
						ObjectType: "StringObject",
						Object:     fmt.Sprintf("test%d", i),
					})
					s.Require().NoError(err)
				}
			}

			if tc.setupCheckpoint != nil {
				err := s.mockStore.StoreCheckpoint(ctx, "test-watcher", *tc.setupCheckpoint)
				s.Require().NoError(err)
			}

			if tc.checkpointErr != nil {
				s.mockStore.WithGetCheckpointInterceptor(func() error {
					return tc.checkpointErr
				})
			}

			if tc.latestErr != nil {
				s.mockStore.WithGetLatestEventInterceptor(func() error {
					return tc.latestErr
				})
			}

			w, err := s.registry.Watch(ctx, "test-watcher", s.mockHandler,
				watcher.WithInitialEventIterator(tc.initialIter))

			if tc.expectedError {
				s.Error(err)
				return
			}
			s.NoError(err)
			s.Equal(tc.expectedIter, w.Stats().NextEventIterator,
				"Iterator mismatch - expected: %s, got: %s",
				tc.expectedIter.String(),
				w.Stats().NextEventIterator.String())
		})
	}
}

func (s *WatcherTestSuite) TestWatcherProcessEvents() {
	ctx := context.Background()
	events := []watcher.StoreEventRequest{
		{Operation: watcher.OperationCreate, ObjectType: "StringObject", Object: "test1"},
		{Operation: watcher.OperationUpdate, ObjectType: "StringObject", Object: "test2"},
	}

	for _, event := range events {
		err := s.mockStore.StoreEvent(ctx, event)
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
	events := []watcher.StoreEventRequest{
		{Operation: watcher.OperationCreate, ObjectType: "StringObject", Object: "test1"},
		{Operation: watcher.OperationUpdate, ObjectType: "StringObject", Object: "test2"},
		{Operation: watcher.OperationDelete, ObjectType: "StringObject", Object: "test3"},
	}

	for _, event := range events {
		err := s.mockStore.StoreEvent(ctx, event)
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
		ObjectTypes: []string{"StringObject"},
		Operations:  []watcher.Operation{watcher.OperationCreate, watcher.OperationUpdate},
	}
	events := []watcher.StoreEventRequest{
		{Operation: watcher.OperationCreate, ObjectType: "StringObject", Object: "test1"},
		{Operation: watcher.OperationUpdate, ObjectType: "StringObject", Object: "test2"},
		{Operation: watcher.OperationDelete, ObjectType: "StringObject", Object: "test3"},
		{Operation: watcher.OperationCreate, ObjectType: "StringObject", Object: "test4"},
		{Operation: watcher.OperationCreate, ObjectType: "OtherStringObject", Object: "test5"},
	}

	for _, event := range events {
		err := s.mockStore.StoreEvent(ctx, event)
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
	events := []watcher.StoreEventRequest{
		{Operation: watcher.OperationCreate, ObjectType: "StringObject", Object: "test1"},
		{Operation: watcher.OperationUpdate, ObjectType: "StringObject", Object: "test2"},
	}

	for _, event := range events {
		err := s.mockStore.StoreEvent(ctx, event)
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
	events := []watcher.StoreEventRequest{
		{Operation: watcher.OperationCreate, ObjectType: "StringObject", Object: "test1"},
		{Operation: watcher.OperationUpdate, ObjectType: "StringObject", Object: "test2"},
		{Operation: watcher.OperationDelete, ObjectType: "StringObject", Object: "test3"},
	}

	for _, event := range events {
		err := s.mockStore.StoreEvent(ctx, event)
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
			events := []watcher.StoreEventRequest{
				{Operation: watcher.OperationCreate, ObjectType: "StringObject", Object: "test1"},
				{Operation: watcher.OperationUpdate, ObjectType: "StringObject", Object: "test2"},
				{Operation: watcher.OperationDelete, ObjectType: "StringObject", Object: "test3"},
				{Operation: watcher.OperationCreate, ObjectType: "StringObject", Object: "test4"},
				{Operation: watcher.OperationUpdate, ObjectType: "StringObject", Object: "test5"},
			}

			for _, event := range events {
				err := s.mockStore.StoreEvent(ctx, event)
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
	events := []watcher.StoreEventRequest{
		{Operation: watcher.OperationCreate, ObjectType: "StringObject", Object: "test1"},
		{Operation: watcher.OperationUpdate, ObjectType: "StringObject", Object: "test2"},
	}

	for _, event := range events {
		err := s.mockStore.StoreEvent(ctx, event)
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
	events := []watcher.StoreEventRequest{
		{Operation: watcher.OperationCreate, ObjectType: "StringObject", Object: "test1"},
		{Operation: watcher.OperationUpdate, ObjectType: "StringObject", Object: "test2"},
		{Operation: watcher.OperationDelete, ObjectType: "StringObject", Object: "test3"},
		{Operation: watcher.OperationCreate, ObjectType: "StringObject", Object: "test4"},
	}

	for _, event := range events {
		err := s.mockStore.StoreEvent(ctx, event)
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
	events := []watcher.StoreEventRequest{
		{Operation: watcher.OperationCreate, ObjectType: "StringObject", Object: "test1"},
		{Operation: watcher.OperationUpdate, ObjectType: "StringObject", Object: "test2"},
	}

	for _, event := range events {
		err := s.mockStore.StoreEvent(ctx, event)
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
				s.Require().NoError(s.mockStore.StoreEvent(ctx, watcher.StoreEventRequest{
					Operation: watcher.OperationCreate, ObjectType: "StringObject", Object: fmt.Sprintf("test%d", i+1),
				}))
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
		events         []watcher.StoreEventRequest
		expectedEvents int
	}{
		{
			name:   "Empty filter",
			filter: watcher.EventFilter{},
			events: []watcher.StoreEventRequest{
				{Operation: watcher.OperationCreate, ObjectType: "StringObject", Object: "test1"},
				{Operation: watcher.OperationUpdate, ObjectType: "OtherStringObject", Object: "test2"},
			},
			expectedEvents: 2,
		},
		{
			name: "Multiple criteria",
			filter: watcher.EventFilter{
				ObjectTypes: []string{"StringObject"},
				Operations:  []watcher.Operation{watcher.OperationCreate},
			},
			events: []watcher.StoreEventRequest{
				{Operation: watcher.OperationCreate, ObjectType: "StringObject", Object: "test1"},
				{Operation: watcher.OperationUpdate, ObjectType: "StringObject", Object: "test2"},
				{Operation: watcher.OperationCreate, ObjectType: "OtherStringObject", Object: "test3"},
			},
			expectedEvents: 1,
		},
		{
			name:   "No matching events",
			filter: watcher.EventFilter{ObjectTypes: []string{"NonExistentType"}},
			events: []watcher.StoreEventRequest{
				{Operation: watcher.OperationCreate, ObjectType: "StringObject", Object: "test1"},
				{Operation: watcher.OperationUpdate, ObjectType: "OtherStringObject", Object: "test2"},
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
				err := s.mockStore.StoreEvent(ctx, event)
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
	events := []watcher.StoreEventRequest{
		{Operation: watcher.OperationCreate, ObjectType: "StringObject", Object: "test1"},
		{Operation: watcher.OperationUpdate, ObjectType: "StringObject", Object: "test2"},
	}

	for _, event := range events {
		err := s.mockStore.StoreEvent(ctx, event)
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
			s.Require().NoError(s.mockStore.StoreEvent(ctx, watcher.StoreEventRequest{
				Operation:  watcher.OperationCreate,
				ObjectType: "StringObject",
				Object:     fmt.Sprintf("test%d", i),
			}))
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
				s.Require().NoError(s.mockStore.StoreEvent(ctx, watcher.StoreEventRequest{
					Operation: watcher.OperationCreate, ObjectType: "StringObject", Object: "test",
				}))
				s.Require().NoError(s.mockStore.StoreEvent(ctx, watcher.StoreEventRequest{
					Operation: watcher.OperationCreate, ObjectType: "StringObject", Object: "test",
				}))
				return nil
			}).Times(1),
		s.mockHandler.EXPECT().HandleEvent(gomock.Any(), watchertest.EventWithSeqNum(2)).Return(nil).Times(1),
		s.mockHandler.EXPECT().HandleEvent(gomock.Any(), watchertest.EventWithSeqNum(3)).Return(nil).Times(1),
		s.mockHandler.EXPECT().HandleEvent(gomock.Any(), watchertest.EventWithSeqNum(4)).Return(nil).Times(1),
	)

	// Store initial events
	s.Require().NoError(s.mockStore.StoreEvent(ctx, watcher.StoreEventRequest{
		Operation: watcher.OperationCreate, ObjectType: "StringObject", Object: "test",
	}))
	s.Require().NoError(s.mockStore.StoreEvent(ctx, watcher.StoreEventRequest{
		Operation: watcher.OperationCreate, ObjectType: "StringObject", Object: "test",
	}))

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

// helper function to get pointer to uint64
func ptr(u uint64) *uint64 {
	return &u
}

func TestWatcherSuite(t *testing.T) {
	suite.Run(t, new(WatcherTestSuite))
}
