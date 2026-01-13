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
	ctx         context.Context
	cancel      context.CancelFunc
	mockStore   *watchertest.EventStoreWrapper
	mockHandler *watcher.MockEventHandler
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

	s.ctx, s.cancel = context.WithTimeout(context.Background(), 5*time.Second)
	s.ctrl = gomock.NewController(s.T())
	s.mockStore = watchertest.NewEventStoreWrapper(boltdbEventStore)
	s.mockHandler = watcher.NewMockEventHandler(s.ctrl)
}

func (s *WatcherTestSuite) TearDownTest() {
	s.ctrl.Finish()
	s.cancel()
}

func (s *WatcherTestSuite) TestCreateWatcher() {
	w, err := watcher.New(s.ctx, "test-watcher", s.mockStore)
	s.Require().NoError(err)
	s.Require().NotNil(w)
	s.Equal("test-watcher", w.ID())

	// Set handler after creation
	err = w.SetHandler(s.mockHandler)
	s.Require().NoError(err)

	// Start watcher async
	s.Require().NoError(w.Start(s.ctx))
	s.Require().Eventually(func() bool { return w.Stats().State == watcher.StateRunning }, 200*time.Millisecond, 10*time.Millisecond)

	// verify stats
	stats := w.Stats()
	s.Equal("test-watcher", stats.ID)
	s.Equal(watcher.StateRunning, stats.State)
	s.Equal(uint64(0), stats.LastProcessedSeqNum)
	s.Equal(time.Time{}, stats.LastProcessedEventTime)

	// Stop the watcher
	w.Stop(s.ctx)
	s.Require().Eventually(func() bool { return w.Stats().State == watcher.StateStopped }, 200*time.Millisecond, 10*time.Millisecond)
}

func (s *WatcherTestSuite) TestSetHandlerErrors() {
	w, err := watcher.New(s.ctx, "test-watcher", s.mockStore)
	s.Require().NoError(err)

	// Test setting nil handler
	err = w.SetHandler(nil)
	s.Error(err)

	// Test setting handler twice
	err = w.SetHandler(s.mockHandler)
	s.NoError(err)
	err = w.SetHandler(s.mockHandler)
	s.Equal(watcher.ErrHandlerExists, err)
}

func (s *WatcherTestSuite) TestStartWithoutHandler() {
	w, err := watcher.New(s.ctx, "test-watcher", s.mockStore)
	s.Require().NoError(err)

	s.Require().Error(w.Start(s.ctx))
	s.Never(func() bool { return w.Stats().State == watcher.StateRunning }, 200*time.Millisecond, 10*time.Millisecond)
}

func (s *WatcherTestSuite) TestDetermineStartingIterator() {

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
			name:             "No checkpoint, non-latest iterator",
			setupLatestEvent: ptr(uint64(10)), // Store event up to seq 15
			initialIter:      watcher.AfterSequenceNumberIterator(5),
			expectedIter:     watcher.AfterSequenceNumberIterator(5),
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
			name:         "Empty store, at iterator",
			initialIter:  watcher.AtSequenceNumberIterator(0),
			expectedIter: watcher.AtSequenceNumberIterator(0),
		},
		{
			name:         "Empty store, after iterator",
			initialIter:  watcher.AfterSequenceNumberIterator(0),
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
		{
			name:             "Sequence at latest",
			setupLatestEvent: ptr(uint64(15)),
			initialIter:      watcher.AtSequenceNumberIterator(15),
			expectedIter:     watcher.AtSequenceNumberIterator(15),
		},

		{
			name:             "Sequence too high, start after latest",
			setupLatestEvent: ptr(uint64(15)),
			initialIter:      watcher.AtSequenceNumberIterator(20),
			expectedIter:     watcher.AfterSequenceNumberIterator(15),
		},
		{
			name:             "After sequence too high, start after latest",
			setupLatestEvent: ptr(uint64(15)),
			initialIter:      watcher.AfterSequenceNumberIterator(20),
			expectedIter:     watcher.AfterSequenceNumberIterator(15),
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
					err := s.mockStore.StoreEvent(s.ctx, watcher.StoreEventRequest{
						Operation:  watcher.OperationCreate,
						ObjectType: "StringObject",
						Object:     fmt.Sprintf("test%d", i),
					})
					s.Require().NoError(err)
				}
			}

			if tc.setupCheckpoint != nil {
				err := s.mockStore.StoreCheckpoint(s.ctx, "test-watcher", *tc.setupCheckpoint)
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

			w, err := watcher.New(s.ctx, "test-watcher", s.mockStore,
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
			s.Equal(tc.expectedIter, w.Stats().CheckpointIterator,
				"Checkpoint iterator mismatch - expected: %s, got: %s",
				tc.expectedIter.String(),
				w.Stats().CheckpointIterator.String())
		})
	}
}

func (s *WatcherTestSuite) TestWatcherProcessEvents() {
	events := []watcher.StoreEventRequest{
		{Operation: watcher.OperationCreate, ObjectType: "StringObject", Object: "test1"},
		{Operation: watcher.OperationUpdate, ObjectType: "StringObject", Object: "test2"},
	}

	for _, event := range events {
		err := s.mockStore.StoreEvent(s.ctx, event)
		s.Require().NoError(err)
	}

	gomock.InOrder(
		s.mockHandler.EXPECT().HandleEvent(gomock.Any(), watchertest.EventWithSeqNum(1)).Return(nil).Times(1),
		s.mockHandler.EXPECT().HandleEvent(gomock.Any(), watchertest.EventWithSeqNum(2)).Return(nil).Times(1),
	)

	w, err := watcher.New(s.ctx, "test-watcher", s.mockStore)
	s.Require().NoError(err)
	s.Require().NoError(w.SetHandler(s.mockHandler))
	s.Require().NoError(w.Start(s.ctx))

	s.waitAndStop(s.ctx, w, 2)
}

func (s *WatcherTestSuite) TestWithStartSeqNum() {
	events := []watcher.StoreEventRequest{
		{Operation: watcher.OperationCreate, ObjectType: "StringObject", Object: "test1"},
		{Operation: watcher.OperationUpdate, ObjectType: "StringObject", Object: "test2"},
		{Operation: watcher.OperationDelete, ObjectType: "StringObject", Object: "test3"},
	}

	for _, event := range events {
		err := s.mockStore.StoreEvent(s.ctx, event)
		s.Require().NoError(err)
	}

	gomock.InOrder(
		s.mockHandler.EXPECT().HandleEvent(gomock.Any(), watchertest.EventWithSeqNum(2)).Return(nil).Times(1),
		s.mockHandler.EXPECT().HandleEvent(gomock.Any(), watchertest.EventWithSeqNum(3)).Return(nil).Times(1),
	)

	w, err := watcher.New(s.ctx, "test-watcher", s.mockStore,
		watcher.WithInitialEventIterator(watcher.AfterSequenceNumberIterator(1)))
	s.Require().NoError(err)
	s.Require().NoError(w.SetHandler(s.mockHandler))

	s.startWaitAndStop(s.ctx, w, 3)
}

func (s *WatcherTestSuite) TestWithFilter() {
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
		err := s.mockStore.StoreEvent(s.ctx, event)
		s.Require().NoError(err)
	}

	gomock.InOrder(
		s.mockHandler.EXPECT().HandleEvent(gomock.Any(), watchertest.EventWithSeqNum(1)).Return(nil).Times(1),
		s.mockHandler.EXPECT().HandleEvent(gomock.Any(), watchertest.EventWithSeqNum(2)).Return(nil).Times(1),
		s.mockHandler.EXPECT().HandleEvent(gomock.Any(), watchertest.EventWithSeqNum(4)).Return(nil).Times(1),
	)

	w, err := watcher.New(s.ctx, "test-watcher", s.mockStore, watcher.WithFilter(filter))
	s.Require().NoError(err)
	s.Require().NoError(w.SetHandler(s.mockHandler))

	s.startWaitAndStop(s.ctx, w, 5)
	s.Equal(uint64(4), w.Stats().LastProcessedSeqNum) // last event not processed
}

func (s *WatcherTestSuite) TestWithHandlerAndAutoStart() {
	s.Run("WithAutoStart requires handler", func() {
		// Should fail because no handler is set
		_, err := watcher.New(s.ctx, "test-watcher", s.mockStore,
			watcher.WithAutoStart())
		s.Require().Error(err)
		s.Contains(err.Error(), "handler must be set when autoStart is enabled")
	})

	s.Run("WithHandler and WithAutoStart starts automatically", func() {
		w, err := watcher.New(s.ctx, "test-watcher", s.mockStore,
			watcher.WithHandler(s.mockHandler),
			watcher.WithAutoStart())
		s.Require().NoError(err)

		// Should be automatically running
		s.Require().Eventually(func() bool {
			return w.Stats().State == watcher.StateRunning
		}, 200*time.Millisecond, 10*time.Millisecond)

		w.Stop(s.ctx)
	})

	s.Run("WithHandler only does not auto-start", func() {
		w, err := watcher.New(s.ctx, "test-watcher", s.mockStore,
			watcher.WithHandler(s.mockHandler))
		s.Require().NoError(err)
		s.Equal(watcher.StateIdle, w.Stats().State)
	})
}

func (s *WatcherTestSuite) TestCheckpoint() {
	events := []watcher.StoreEventRequest{
		{Operation: watcher.OperationCreate, ObjectType: "StringObject", Object: "test1"},
		{Operation: watcher.OperationUpdate, ObjectType: "StringObject", Object: "test2"},
	}

	for _, event := range events {
		err := s.mockStore.StoreEvent(s.ctx, event)
		s.Require().NoError(err)
	}

	gomock.InOrder(
		s.mockHandler.EXPECT().HandleEvent(gomock.Any(), watchertest.EventWithSeqNum(1)).Return(nil).Times(1),
		s.mockHandler.EXPECT().HandleEvent(gomock.Any(), watchertest.EventWithSeqNum(2)).Return(nil).Times(1),
	)
	w, err := watcher.New(s.ctx, "test-watcher", s.mockStore)
	s.Require().NoError(err)
	s.Require().NoError(w.SetHandler(s.mockHandler))

	s.startAndWait(s.ctx, w, 2)

	// Manually checkpoint
	err = w.Checkpoint(s.ctx, 1)
	s.Require().NoError(err)

	// Verify both checkpoint and checkpointIterator
	checkpoint, err := s.mockStore.GetCheckpoint(s.ctx, w.ID())
	s.Require().NoError(err)
	s.Equal(uint64(1), checkpoint)
	s.Equal(watcher.AfterSequenceNumberIterator(1), w.Stats().CheckpointIterator)

	w.Stop(s.ctx)

	// Start a new watcher and verify it starts from checkpoint
	newHandler := watcher.NewMockEventHandler(s.ctrl)
	newHandler.EXPECT().HandleEvent(gomock.Any(), watchertest.EventWithSeqNum(2)).Return(nil).Times(1)

	w, err = watcher.New(s.ctx, "test-watcher", s.mockStore)
	s.Require().NoError(err)
	s.Require().NoError(w.SetHandler(newHandler))

	// Verify both iterators start at checkpoint
	s.Equal(watcher.AfterSequenceNumberIterator(1), w.Stats().CheckpointIterator)
	s.Equal(watcher.AfterSequenceNumberIterator(1), w.Stats().NextEventIterator)

	s.startWaitAndStop(s.ctx, w, 2)
}

func (s *WatcherTestSuite) TestRestartFromCheckpoint() {
	events := []watcher.StoreEventRequest{
		{Operation: watcher.OperationCreate, ObjectType: "StringObject", Object: "test1"},
		{Operation: watcher.OperationUpdate, ObjectType: "StringObject", Object: "test2"},
	}

	for _, event := range events {
		err := s.mockStore.StoreEvent(s.ctx, event)
		s.Require().NoError(err)
	}

	// First start and process events
	gomock.InOrder(
		s.mockHandler.EXPECT().HandleEvent(gomock.Any(), watchertest.EventWithSeqNum(1)).Return(nil).Times(1),
		s.mockHandler.EXPECT().HandleEvent(gomock.Any(), watchertest.EventWithSeqNum(2)).Return(nil).Times(1),
	)

	w, err := watcher.New(s.ctx, "test-watcher", s.mockStore)
	s.Require().NoError(err)
	s.Require().NoError(w.SetHandler(s.mockHandler))
	s.Require().NoError(w.Start(s.ctx))

	// Wait and checkpoint at 1
	s.wait(s.ctx, w, 2)
	s.Require().NoError(w.Checkpoint(s.ctx, 1))
	w.Stop(s.ctx)

	// Restart and verify it starts from checkpoint
	s.mockHandler.EXPECT().HandleEvent(gomock.Any(), watchertest.EventWithSeqNum(2)).Return(nil).Times(1)

	s.Require().NoError(w.Start(s.ctx))
	s.Equal(watcher.AfterSequenceNumberIterator(1), w.Stats().NextEventIterator)
	s.Equal(watcher.AfterSequenceNumberIterator(1), w.Stats().CheckpointIterator)

	s.waitAndStop(s.ctx, w, 2)
}

func (s *WatcherTestSuite) TestSeekToOffset() {
	events := []watcher.StoreEventRequest{
		{Operation: watcher.OperationCreate, ObjectType: "StringObject", Object: "test1"},
		{Operation: watcher.OperationUpdate, ObjectType: "StringObject", Object: "test2"},
		{Operation: watcher.OperationDelete, ObjectType: "StringObject", Object: "test3"},
	}

	for _, event := range events {
		err := s.mockStore.StoreEvent(s.ctx, event)
		s.Require().NoError(err)
	}

	gomock.InOrder(
		s.mockHandler.EXPECT().HandleEvent(gomock.Any(), watchertest.EventWithSeqNum(1)).Return(nil).Times(1),
		s.mockHandler.EXPECT().HandleEvent(gomock.Any(), watchertest.EventWithSeqNum(2)).Return(nil).Times(1),

		// last event is processed twice after seek
		s.mockHandler.EXPECT().HandleEvent(gomock.Any(), watchertest.EventWithSeqNum(3)).Return(nil).Times(2),
	)
	w, err := watcher.New(s.ctx, "test-watcher", s.mockStore)
	s.Require().NoError(err)
	s.Require().NoError(w.SetHandler(s.mockHandler))
	s.startAndWait(s.ctx, w, 3)

	// Seek to offset 2
	s.Require().NoError(w.SeekToOffset(s.ctx, 2))

	// Verify the checkpoint
	checkpoint, err := s.mockStore.GetCheckpoint(s.ctx, w.ID())
	s.Require().NoError(err)
	s.Equal(uint64(2), checkpoint)

	s.waitAndStop(s.ctx, w, 3)
}

func (s *WatcherTestSuite) TestCheckpointAndStartSeqNum() {
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
				err := s.mockStore.StoreEvent(s.ctx, event)
				s.Require().NoError(err)
			}

			// Set up the checkpoint if provided
			if tc.checkpoint != 0 {
				err := s.mockStore.StoreCheckpoint(s.ctx, "test-watcher", tc.checkpoint)
				s.Require().NoError(err)
			}

			// Set up expectations
			for _, expectedSeqNum := range tc.expectedEvents {
				s.mockHandler.EXPECT().HandleEvent(gomock.Any(), watchertest.EventWithSeqNum(expectedSeqNum)).Return(nil).Times(1)
			}

			// Start the watcher
			w, err := watcher.New(s.ctx, "test-watcher", s.mockStore,
				watcher.WithInitialEventIterator(watcher.AfterSequenceNumberIterator(tc.startSeqNum)))
			s.Require().NoError(err)
			s.Require().NoError(w.SetHandler(s.mockHandler))

			// Wait for processing and stop
			s.startWaitAndStop(s.ctx, w, uint64(5))
		})
	}
}

func (s *WatcherTestSuite) TestHandleEventErrorWithBlockStrategy() {
	events := []watcher.StoreEventRequest{
		{Operation: watcher.OperationCreate, ObjectType: "StringObject", Object: "test1"},
		{Operation: watcher.OperationUpdate, ObjectType: "StringObject", Object: "test2"},
	}

	for _, event := range events {
		err := s.mockStore.StoreEvent(s.ctx, event)
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

	w, err := watcher.New(s.ctx, "test-watcher", s.mockStore,
		watcher.WithMaxRetries(3), // will be ignored with block strategy
		watcher.WithInitialBackoff(1*time.Nanosecond),
		watcher.WithMaxBackoff(1*time.Nanosecond),
		watcher.WithRetryStrategy(watcher.RetryStrategyBlock))
	s.Require().NoError(err)
	s.Require().NoError(w.SetHandler(s.mockHandler))

	s.startWaitAndStop(s.ctx, w, 2)
	s.Equal(uint64(2), w.Stats().LastProcessedSeqNum)
	s.Equal(maxFails+1, failCount) // Verify that it failed 100 times before succeeding
}

func (s *WatcherTestSuite) TestDifferentIteratorTypes() {
	events := []watcher.StoreEventRequest{
		{Operation: watcher.OperationCreate, ObjectType: "StringObject", Object: "test1"},
		{Operation: watcher.OperationUpdate, ObjectType: "StringObject", Object: "test2"},
		{Operation: watcher.OperationDelete, ObjectType: "StringObject", Object: "test3"},
		{Operation: watcher.OperationCreate, ObjectType: "StringObject", Object: "test4"},
	}

	for _, event := range events {
		err := s.mockStore.StoreEvent(s.ctx, event)
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

			w, err := watcher.New(s.ctx, "test-watcher", s.mockStore,
				watcher.WithInitialEventIterator(tc.iterator))
			s.Require().NoError(err)
			s.Require().NoError(w.SetHandler(s.mockHandler))

			s.startWaitAndStop(s.ctx, w, 4)
		})
	}
}

func (s *WatcherTestSuite) TestEmptyEventStoreWithDifferentIterators() {

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
			expectedIterator: watcher.AfterSequenceNumberIterator(0),
		},
		{
			name:             "AfterSequenceNumber(0)",
			iterator:         watcher.AfterSequenceNumberIterator(0),
			expectedIterator: watcher.AfterSequenceNumberIterator(0),
		},
		{
			name:             "AfterSequenceNumber(1)",
			iterator:         watcher.AfterSequenceNumberIterator(1),
			expectedIterator: watcher.AfterSequenceNumberIterator(0),
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.mockHandler = watcher.NewMockEventHandler(s.ctrl)

			// Test with empty store
			w, err := watcher.New(s.ctx, "test-watcher", s.mockStore,
				watcher.WithInitialEventIterator(tc.iterator))
			s.Require().NoError(err)
			s.Require().NoError(w.SetHandler(s.mockHandler))
			s.Require().NoError(w.Start(s.ctx))

			// Wait for the watcher to process and convert the iterator to the expected type.
			// This is necessary because the watcher may start with a different iterator type
			// (e.g., TrimHorizon) and convert it after the first GetEvents call.
			s.Require().Eventually(func() bool {
				stats := w.Stats()
				return stats.State == watcher.StateRunning &&
					stats.NextEventIterator == tc.expectedIterator
			}, 1*time.Second, 10*time.Millisecond)

			w.Stop(s.ctx)
			s.Require().Eventually(func() bool {
				return w.Stats().State == watcher.StateStopped
			}, 1*time.Second, 10*time.Millisecond)

			// Check if the next event iterator matches the expected iterator
			s.Equal(tc.expectedIterator, w.Stats().NextEventIterator)
		})
	}
}

func (s *WatcherTestSuite) TestHandleEventErrorWithSkipStrategy() {
	events := []watcher.StoreEventRequest{
		{Operation: watcher.OperationCreate, ObjectType: "StringObject", Object: "test1"},
		{Operation: watcher.OperationUpdate, ObjectType: "StringObject", Object: "test2"},
	}

	for _, event := range events {
		err := s.mockStore.StoreEvent(s.ctx, event)
		s.Require().NoError(err)
	}

	gomock.InOrder(
		s.mockHandler.EXPECT().HandleEvent(gomock.Any(), watchertest.EventWithSeqNum(1)).Return(errors.New("handling error")).Times(3),
		s.mockHandler.EXPECT().HandleEvent(gomock.Any(), watchertest.EventWithSeqNum(2)).Return(nil).Times(1),
	)

	w, err := watcher.New(s.ctx, "test-watcher", s.mockStore,
		watcher.WithMaxRetries(3),
		watcher.WithInitialBackoff(1*time.Nanosecond),
		watcher.WithRetryStrategy(watcher.RetryStrategySkip))
	s.Require().NoError(err)
	s.Require().NoError(w.SetHandler(s.mockHandler))

	s.startWaitAndStop(s.ctx, w, 2)
	s.Equal(uint64(2), w.Stats().LastProcessedSeqNum)
}

func (s *WatcherTestSuite) TestBatchOptions() {

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
				s.Require().NoError(s.mockStore.StoreEvent(s.ctx, watcher.StoreEventRequest{
					Operation: watcher.OperationCreate, ObjectType: "StringObject", Object: fmt.Sprintf("test%d", i+1),
				}))
			}

			getEventsCallCount := 0
			s.mockStore.WithGetEventsInterceptor(func() error {
				getEventsCallCount++
				return nil
			})

			w, err := watcher.New(s.ctx, "test-watcher", s.mockStore,
				watcher.WithBatchSize(tc.batchSize),
			)
			s.Require().NoError(err)
			s.Require().NoError(w.SetHandler(s.mockHandler))

			s.mockHandler.EXPECT().HandleEvent(gomock.Any(), gomock.Any()).Return(nil).Times(tc.eventCount)

			s.startWaitAndStop(s.ctx, w, uint64(tc.eventCount))
			s.Equal(tc.expectedCalls+1, getEventsCallCount) // one extra call longpolling for new events
		})
	}
}

func (s *WatcherTestSuite) TestFilterEdgeCases() {

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
				err := s.mockStore.StoreEvent(s.ctx, event)
				s.Require().NoError(err)
			}

			w, err := watcher.New(s.ctx, "test-watcher", s.mockStore,
				watcher.WithFilter(tc.filter))
			s.Require().NoError(err)
			s.Require().NoError(w.SetHandler(s.mockHandler))

			s.mockHandler.EXPECT().HandleEvent(gomock.Any(), gomock.Any()).
				Return(nil).Times(tc.expectedEvents)

			s.startWaitAndStop(s.ctx, w, uint64(len(tc.events)))
			s.Equal(uint64(tc.expectedEvents), w.Stats().LastProcessedSeqNum)
		})
	}
}

func (s *WatcherTestSuite) TestRestartBehavior() {
	events := []watcher.StoreEventRequest{
		{Operation: watcher.OperationCreate, ObjectType: "StringObject", Object: "test1"},
		{Operation: watcher.OperationUpdate, ObjectType: "StringObject", Object: "test2"},
	}

	for _, event := range events {
		err := s.mockStore.StoreEvent(s.ctx, event)
		s.Require().NoError(err)
	}

	for i := 0; i < 3; i++ {
		handler := watcher.NewMockEventHandler(s.ctrl)
		gomock.InOrder(
			handler.EXPECT().HandleEvent(gomock.Any(), watchertest.EventWithSeqNum(1)).Return(nil).Times(1),
			handler.EXPECT().HandleEvent(gomock.Any(), watchertest.EventWithSeqNum(2)).Return(nil).Times(1),
		)

		w, err := watcher.New(s.ctx, "test-watcher", s.mockStore)
		s.Require().NoError(err)

		s.Require().NoError(w.SetHandler(handler))
		s.startWaitAndStop(s.ctx, w, 2)
	}
}

func (s *WatcherTestSuite) TestListenAndStoreConcurrently() {

	w, err := watcher.New(s.ctx, "test-watcher", s.mockStore,
		watcher.WithBatchSize(7))
	s.Require().NoError(err)
	s.Require().NoError(w.SetHandler(s.mockHandler))
	s.startAndWait(s.ctx, w, 0)

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
			s.Require().NoError(s.mockStore.StoreEvent(s.ctx, watcher.StoreEventRequest{
				Operation:  watcher.OperationCreate,
				ObjectType: "StringObject",
				Object:     fmt.Sprintf("test%d", i),
			}))
		}
		close(done)
	}()

	<-done
	gomock.InOrder(expectations...)

	s.waitAndStop(s.ctx, w, uint64(eventCount))
}

func (s *WatcherTestSuite) TestEventStoreConsistency() {

	w, err := watcher.New(s.ctx, "test-watcher", s.mockStore,
		watcher.WithBatchSize(2))
	s.Require().NoError(err)
	s.Require().NoError(w.SetHandler(s.mockHandler))
	s.startAndWait(s.ctx, w, 0)

	// Set up expectations
	gomock.InOrder(
		s.mockHandler.EXPECT().HandleEvent(gomock.Any(), watchertest.EventWithSeqNum(1)).DoAndReturn(
			func(ctx context.Context, event watcher.Event) error {
				// Store more events while processing
				s.Require().NoError(s.mockStore.StoreEvent(s.ctx, watcher.StoreEventRequest{
					Operation: watcher.OperationCreate, ObjectType: "StringObject", Object: "test",
				}))
				s.Require().NoError(s.mockStore.StoreEvent(s.ctx, watcher.StoreEventRequest{
					Operation: watcher.OperationCreate, ObjectType: "StringObject", Object: "test",
				}))
				return nil
			}).Times(1),
		s.mockHandler.EXPECT().HandleEvent(gomock.Any(), watchertest.EventWithSeqNum(2)).Return(nil).Times(1),
		s.mockHandler.EXPECT().HandleEvent(gomock.Any(), watchertest.EventWithSeqNum(3)).Return(nil).Times(1),
		s.mockHandler.EXPECT().HandleEvent(gomock.Any(), watchertest.EventWithSeqNum(4)).Return(nil).Times(1),
	)

	// Store initial events
	s.Require().NoError(s.mockStore.StoreEvent(s.ctx, watcher.StoreEventRequest{
		Operation: watcher.OperationCreate, ObjectType: "StringObject", Object: "test",
	}))
	s.Require().NoError(s.mockStore.StoreEvent(s.ctx, watcher.StoreEventRequest{
		Operation: watcher.OperationCreate, ObjectType: "StringObject", Object: "test",
	}))

	s.waitAndStop(s.ctx, w, 4)
}
func (s *WatcherTestSuite) TestEphemeralWatcherBasic() {
	events := []watcher.StoreEventRequest{
		{Operation: watcher.OperationCreate, ObjectType: "StringObject", Object: "test1"},
		{Operation: watcher.OperationUpdate, ObjectType: "StringObject", Object: "test2"},
	}

	for _, event := range events {
		err := s.mockStore.StoreEvent(s.ctx, event)
		s.Require().NoError(err)
	}

	// Create ephemeral watcher
	w, err := watcher.New(s.ctx, "test-watcher", s.mockStore,
		watcher.WithEphemeral())
	s.Require().NoError(err)
	s.Require().NoError(w.SetHandler(s.mockHandler))

	// Verify events are processed
	gomock.InOrder(
		s.mockHandler.EXPECT().HandleEvent(gomock.Any(), watchertest.EventWithSeqNum(1)).Return(nil).Times(1),
		s.mockHandler.EXPECT().HandleEvent(gomock.Any(), watchertest.EventWithSeqNum(2)).Return(nil).Times(1),
	)

	s.startWaitAndStop(s.ctx, w, 2)
}

func (s *WatcherTestSuite) TestEphemeralWatcherIgnoresCheckpoint() {
	// Store some events
	events := []watcher.StoreEventRequest{
		{Operation: watcher.OperationCreate, ObjectType: "StringObject", Object: "test1"},
		{Operation: watcher.OperationUpdate, ObjectType: "StringObject", Object: "test2"},
		{Operation: watcher.OperationDelete, ObjectType: "StringObject", Object: "test3"},
	}

	for _, event := range events {
		err := s.mockStore.StoreEvent(s.ctx, event)
		s.Require().NoError(err)
	}

	// Store a checkpoint
	err := s.mockStore.StoreCheckpoint(s.ctx, "test-watcher", 2)
	s.Require().NoError(err)

	// Create ephemeral watcher - should start from beginning despite checkpoint
	w, err := watcher.New(s.ctx, "test-watcher", s.mockStore,
		watcher.WithEphemeral(),
		watcher.WithInitialEventIterator(watcher.TrimHorizonIterator()))
	s.Require().NoError(err)
	s.Require().NoError(w.SetHandler(s.mockHandler))

	// Should process all events from the beginning
	gomock.InOrder(
		s.mockHandler.EXPECT().HandleEvent(gomock.Any(), watchertest.EventWithSeqNum(1)).Return(nil).Times(1),
		s.mockHandler.EXPECT().HandleEvent(gomock.Any(), watchertest.EventWithSeqNum(2)).Return(nil).Times(1),
		s.mockHandler.EXPECT().HandleEvent(gomock.Any(), watchertest.EventWithSeqNum(3)).Return(nil).Times(1),
	)

	s.startWaitAndStop(s.ctx, w, 3)
}

func (s *WatcherTestSuite) TestEphemeralWatcherCheckpointFails() {
	w, err := watcher.New(s.ctx, "test-watcher", s.mockStore,
		watcher.WithEphemeral())
	s.Require().NoError(err)

	// Attempt to checkpoint should fail
	err = w.Checkpoint(s.ctx, 1)
	s.Require().Error(err)
	s.Contains(err.Error(), "cannot checkpoint ephemeral watcher")

	// Verify no checkpoint was stored
	_, err = s.mockStore.GetCheckpoint(s.ctx, "test-watcher")
	s.Require().Error(err)
	s.True(errors.Is(err, watcher.ErrCheckpointNotFound))
}

func (s *WatcherTestSuite) TestEphemeralWatcherSeek() {
	events := []watcher.StoreEventRequest{
		{Operation: watcher.OperationCreate, ObjectType: "StringObject", Object: "test1"},
		{Operation: watcher.OperationUpdate, ObjectType: "StringObject", Object: "test2"},
		{Operation: watcher.OperationDelete, ObjectType: "StringObject", Object: "test3"},
	}

	for _, event := range events {
		err := s.mockStore.StoreEvent(s.ctx, event)
		s.Require().NoError(err)
	}

	gomock.InOrder(
		s.mockHandler.EXPECT().HandleEvent(gomock.Any(), watchertest.EventWithSeqNum(1)).Return(nil).Times(1),
		s.mockHandler.EXPECT().HandleEvent(gomock.Any(), watchertest.EventWithSeqNum(2)).Return(nil).Times(1),

		// last event is processed twice after seek
		s.mockHandler.EXPECT().HandleEvent(gomock.Any(), watchertest.EventWithSeqNum(3)).Return(nil).Times(2),
	)
	w, err := watcher.New(s.ctx, "test-watcher", s.mockStore, watcher.WithEphemeral())
	s.Require().NoError(err)
	s.Require().NoError(w.SetHandler(s.mockHandler))
	s.startAndWait(s.ctx, w, 3)

	// Seek to offset 2
	s.Require().NoError(w.SeekToOffset(s.ctx, 2))

	// Verify no checkpoint
	_, err = s.mockStore.GetCheckpoint(s.ctx, w.ID())
	s.Require().Error(err)

	s.waitAndStop(s.ctx, w, 3)
}

func (s *WatcherTestSuite) TestEphemeralWatcherRestart() {
	events := []watcher.StoreEventRequest{
		{Operation: watcher.OperationCreate, ObjectType: "StringObject", Object: "test1"},
		{Operation: watcher.OperationUpdate, ObjectType: "StringObject", Object: "test2"},
	}

	for _, event := range events {
		err := s.mockStore.StoreEvent(s.ctx, event)
		s.Require().NoError(err)
	}

	w, err := watcher.New(s.ctx, "test-watcher", s.mockStore,
		watcher.WithEphemeral(),
		watcher.WithInitialEventIterator(watcher.TrimHorizonIterator()))
	s.Require().NoError(err)
	s.Require().NoError(w.SetHandler(s.mockHandler))

	// First start and process events
	gomock.InOrder(
		s.mockHandler.EXPECT().HandleEvent(gomock.Any(), watchertest.EventWithSeqNum(1)).Return(nil).Times(1),
		s.mockHandler.EXPECT().HandleEvent(gomock.Any(), watchertest.EventWithSeqNum(2)).Return(nil).Times(1),
	)

	s.startWaitAndStop(s.ctx, w, 2)

	// Restart and verify it starts from beginning again
	s.mockHandler.EXPECT().HandleEvent(gomock.Any(), watchertest.EventWithSeqNum(1)).Return(nil).Times(1)
	s.mockHandler.EXPECT().HandleEvent(gomock.Any(), watchertest.EventWithSeqNum(2)).Return(nil).Times(1)

	s.Require().NoError(w.Start(s.ctx))
	s.waitAndStop(s.ctx, w, 2)
}

func (s *WatcherTestSuite) TestEphemeralVsNonEphemeralRestart() {
	events := []watcher.StoreEventRequest{
		{Operation: watcher.OperationCreate, ObjectType: "StringObject", Object: "test1"},
		{Operation: watcher.OperationUpdate, ObjectType: "StringObject", Object: "test2"},
	}

	for _, event := range events {
		err := s.mockStore.StoreEvent(s.ctx, event)
		s.Require().NoError(err)
	}

	// First with regular watcher
	regularWatcher, err := watcher.New(s.ctx, "test-watcher", s.mockStore)
	s.Require().NoError(err)
	s.Require().NoError(regularWatcher.SetHandler(s.mockHandler))

	gomock.InOrder(
		s.mockHandler.EXPECT().HandleEvent(gomock.Any(), watchertest.EventWithSeqNum(1)).Return(nil).Times(1),
		s.mockHandler.EXPECT().HandleEvent(gomock.Any(), watchertest.EventWithSeqNum(2)).Return(nil).Times(1),
	)

	s.startAndWait(s.ctx, regularWatcher, 2)
	err = regularWatcher.Checkpoint(s.ctx, 2)
	s.Require().NoError(err)
	regularWatcher.Stop(s.ctx)

	// Then with ephemeral watcher using same ID
	ephemeralWatcher, err := watcher.New(s.ctx, "test-watcher", s.mockStore,
		watcher.WithEphemeral())
	s.Require().NoError(err)
	s.Require().NoError(ephemeralWatcher.SetHandler(s.mockHandler))

	// Should process all events despite existing checkpoint
	gomock.InOrder(
		s.mockHandler.EXPECT().HandleEvent(gomock.Any(), watchertest.EventWithSeqNum(1)).Return(nil).Times(1),
		s.mockHandler.EXPECT().HandleEvent(gomock.Any(), watchertest.EventWithSeqNum(2)).Return(nil).Times(1),
	)

	s.startWaitAndStop(s.ctx, ephemeralWatcher, 2)
}

func (s *WatcherTestSuite) TestStopStates() {

	s.Run("stop running watcher", func() {
		w, err := watcher.New(s.ctx, "test-watcher", s.mockStore)
		s.Require().NoError(err)
		s.Require().NoError(w.SetHandler(s.mockHandler))
		s.startAndWait(s.ctx, w, 0)

		w.Stop(s.ctx)
		s.Equal(watcher.StateStopped, w.Stats().State)
	})

	s.Run("stop stopped watcher", func() {
		w, err := watcher.New(s.ctx, "test-watcher", s.mockStore)
		s.Require().NoError(err)
		s.Require().NoError(w.SetHandler(s.mockHandler))
		s.startAndWait(s.ctx, w, 0)

		w.Stop(s.ctx)
		s.Equal(watcher.StateStopped, w.Stats().State)

		// Stop again
		w.Stop(s.ctx)
		s.Equal(watcher.StateStopped, w.Stats().State)
	})

	s.Run("stop not-started watcher", func() {
		w, err := watcher.New(s.ctx, "test-watcher", s.mockStore)
		s.Require().NoError(err)
		s.Equal(watcher.StateIdle, w.Stats().State)

		w.Stop(s.ctx)
		s.Equal(watcher.StateStopped, w.Stats().State)
	})

	s.Run("concurrent stop calls", func() {
		// Create a channel to control GetEvents
		getEventsCh := make(chan struct{})

		// Set up the mockStore to block on GetEvents
		s.mockStore.WithGetEventsInterceptor(func() error {
			<-getEventsCh
			return nil
		})

		w, err := watcher.New(s.ctx, "test-watcher", s.mockStore)
		s.Require().NoError(err)
		s.Require().NoError(w.SetHandler(s.mockHandler))
		s.startAndWait(s.ctx, w, 0)

		// Start first stop operation - will be blocked due to GetEvents
		go w.Stop(s.ctx)

		// Wait for watcher to enter stopping state
		s.Eventually(func() bool {
			return w.Stats().State == watcher.StateStopping
		}, 200*time.Millisecond, 10*time.Millisecond)

		// Try second stop while first one is still in progress
		go w.Stop(s.ctx)

		// State should still be stopping
		s.Eventually(func() bool {
			return w.Stats().State == watcher.StateStopping
		}, 200*time.Millisecond, 10*time.Millisecond)

		// Unblock GetEvents
		close(getEventsCh)

		// Now watcher should transition to stopped
		s.Eventually(func() bool {
			return w.Stats().State == watcher.StateStopped
		}, 200*time.Millisecond, 10*time.Millisecond)
	})
}

func (s *WatcherTestSuite) wait(ctx context.Context, w watcher.Watcher, continuationSeqNum uint64) {
	s.Require().Eventually(func() bool {
		return w.Stats().State == watcher.StateRunning &&
			w.Stats().NextEventIterator.SequenceNumber == continuationSeqNum
	}, 1*time.Second, 10*time.Millisecond)
}

// startWaitAndStop
func (s *WatcherTestSuite) startAndWait(ctx context.Context, w watcher.Watcher, continuationSeqNum uint64) {
	s.Require().NoError(w.Start(s.ctx))
	s.wait(s.ctx, w, continuationSeqNum)
}

// startWaitAndStop
func (s *WatcherTestSuite) startWaitAndStop(ctx context.Context, w watcher.Watcher, continuationSeqNum uint64) {
	s.Require().NoError(w.Start(s.ctx))
	s.waitAndStop(s.ctx, w, continuationSeqNum)
}

func (s *WatcherTestSuite) waitAndStop(ctx context.Context, w watcher.Watcher, continuationSeqNum uint64) {
	s.wait(s.ctx, w, continuationSeqNum)
	w.Stop(s.ctx)
	s.Require().Eventually(func() bool { return w.Stats().State == watcher.StateStopped }, 1*time.Second, 10*time.Millisecond)
}

// helper function to get pointer to uint64
func ptr(u uint64) *uint64 {
	return &u
}

func TestWatcherSuite(t *testing.T) {
	suite.Run(t, new(WatcherTestSuite))
}
