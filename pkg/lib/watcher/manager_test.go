//go:build unit || !integration

package watcher_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/bacalhau-project/bacalhau/pkg/lib/watcher"
	"github.com/bacalhau-project/bacalhau/pkg/lib/watcher/boltdb"
	watchertest "github.com/bacalhau-project/bacalhau/pkg/lib/watcher/test"
)

type ManagerTestSuite struct {
	suite.Suite
	ctrl        *gomock.Controller
	mockStore   *watchertest.EventStoreWrapper
	mockHandler *watcher.MockEventHandler
	manager     watcher.Manager
}

func (s *ManagerTestSuite) SetupTest() {
	boltdbEventStore, err := boltdb.NewEventStore(watchertest.CreateBoltDB(s.T()))
	s.Require().NoError(err)

	s.ctrl = gomock.NewController(s.T())
	s.mockStore = watchertest.NewEventStoreWrapper(boltdbEventStore)
	s.mockHandler = watcher.NewMockEventHandler(s.ctrl)
	s.manager = watcher.NewManager(s.mockStore)
}

func (s *ManagerTestSuite) TearDownTest() {
	s.Require().NoError(s.manager.Stop(context.Background()), "failed to stop manager in teardown")
	s.ctrl.Finish()
}

func (s *ManagerTestSuite) TestCreate() {
	ctx := context.Background()
	watcherID := "test-watcher"

	// Create watcher
	w, err := s.manager.Create(ctx, watcherID)
	s.Require().NoError(err)
	s.Require().NotNil(w)
	s.Equal(watcherID, w.ID())

	// Set handler and start watcher
	err = w.SetHandler(s.mockHandler)
	s.Require().NoError(err)
	s.startAndWait(ctx, w)

	// Stop the manager and ensure the watcher is stopped
	err = s.manager.Stop(ctx)
	s.Require().NoError(err)
	s.Require().Equal(watcher.StateStopped, w.Stats().State)
}

func (s *ManagerTestSuite) TestCreateDuplicateWatcher() {
	ctx := context.Background()
	watcherID := "test-watcher"

	_, err := s.manager.Create(ctx, watcherID)
	s.Require().NoError(err)

	// Try to create another watcher with same ID
	_, err = s.manager.Create(ctx, watcherID)
	s.Require().Error(err)
	s.Contains(err.Error(), "watcher already exists")
}

func (s *ManagerTestSuite) TestLookup() {
	ctx := context.Background()
	watcherID := "test-watcher"

	w, err := s.manager.Create(ctx, watcherID)
	s.Require().NoError(err)

	// Lookup existing watcher
	retrievedWatcher, err := s.manager.Lookup(watcherID)
	s.Require().NoError(err)
	s.Require().NotNil(retrievedWatcher)
	s.Equal(watcherID, retrievedWatcher.ID())
	s.Equal(w, retrievedWatcher)
}

func (s *ManagerTestSuite) TestLookupNonExistentWatcher() {
	_, err := s.manager.Lookup("non-existent")
	s.Require().Error(err)
	s.Contains(err.Error(), "watcher not found")
}

func (s *ManagerTestSuite) TestStop() {
	ctx := context.Background()

	// create a started watcher
	w1, err := s.manager.Create(ctx, "watcher-1")
	s.Require().NoError(err)
	s.Require().NoError(w1.SetHandler(s.mockHandler))
	s.startAndWait(ctx, w1)

	// create a stopped watcher
	w2, err := s.manager.Create(ctx, "watcher-2")
	s.Require().NoError(err)
	s.Require().NoError(w2.SetHandler(s.mockHandler))
	s.startAndWait(ctx, w2)
	w2.Stop(ctx)

	// create a non-started watcher
	w3, err := s.manager.Create(ctx, "watcher-3")
	s.Require().NoError(err)

	err = s.manager.Stop(ctx)
	s.Require().NoError(err)
	s.Require().Equal(watcher.StateStopped, w1.Stats().State)
	s.Require().Equal(watcher.StateStopped, w2.Stats().State)
	s.Require().Equal(watcher.StateStopped, w3.Stats().State)
}

func (s *ManagerTestSuite) TestStopWithTimeout() {
	ctx := context.Background()
	watcherID := "test-watcher"

	// Create a channel to control GetEvents
	getEventsCh := make(chan struct{})

	//// Set up the mockStore to block on GetEvents
	s.mockStore.WithGetEventsInterceptor(func() error {
		<-getEventsCh
		return nil
	})

	w, err := s.manager.Create(ctx, watcherID)
	s.Require().NoError(err)
	s.Require().NoError(w.SetHandler(s.mockHandler))
	s.startAndWait(ctx, w)

	// Create a very short timeout
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 1*time.Nanosecond)
	defer cancel()

	err = s.manager.Stop(ctxWithTimeout)
	s.Require().Error(err)
	s.Equal(context.DeadlineExceeded, err)

	// Ensure the watcher is stopping
	s.Require().Eventually(func() bool {
		return w.Stats().State == watcher.StateStopping
	}, 200*time.Millisecond, 10*time.Millisecond)

	// verify that the watcher is still stopping
	time.Sleep(100 * time.Millisecond)
	s.Require().Equal(watcher.StateStopping, w.Stats().State)

	// Unblock GetEvents
	close(getEventsCh)

	// Ensure the watcher is stopped
	s.Require().Eventually(func() bool {
		return w.Stats().State == watcher.StateStopped
	}, 200*time.Millisecond, 10*time.Millisecond)
}

func (s *ManagerTestSuite) TestWatcherProcessesEvents() {
	ctx := context.Background()
	watcherID := "test-watcher"

	events := []watcher.StoreEventRequest{
		{Operation: watcher.OperationCreate, ObjectType: "TestObject", Object: "test1"},
		{Operation: watcher.OperationUpdate, ObjectType: "TestObject", Object: "test2"},
	}

	for _, event := range events {
		err := s.mockStore.StoreEvent(ctx, event)
		s.Require().NoError(err)
	}

	s.mockHandler.EXPECT().HandleEvent(gomock.Any(), gomock.Any()).Return(nil).Times(2)

	w, err := s.manager.Create(ctx, watcherID)
	s.Require().NoError(err)
	s.Require().NoError(w.SetHandler(s.mockHandler))
	s.startAndWait(ctx, w)

	// Wait for events to be processed
	s.Require().Eventually(func() bool {
		return w.Stats().LastProcessedSeqNum == 2
	}, 200*time.Millisecond, 10*time.Millisecond)

	err = s.manager.Stop(ctx)
	s.Require().NoError(err)
}

func (s *ManagerTestSuite) TestMultipleWatchers() {
	ctx := context.Background()
	watcherID1 := "test-watcher-1"
	watcherID2 := "test-watcher-2"

	events := []watcher.StoreEventRequest{
		{Operation: watcher.OperationCreate, ObjectType: "TestObject", Object: "test1"},
		{Operation: watcher.OperationUpdate, ObjectType: "TestObject", Object: "test2"},
	}

	for _, event := range events {
		err := s.mockStore.StoreEvent(ctx, event)
		s.Require().NoError(err)
	}

	s.mockHandler.EXPECT().HandleEvent(gomock.Any(), gomock.Any()).Return(nil).Times(4)

	// Create and start first watcher
	w1, err := s.manager.Create(ctx, watcherID1)
	s.Require().NoError(err)
	s.Require().NoError(w1.SetHandler(s.mockHandler))
	s.startAndWait(ctx, w1)

	// Create and start second watcher
	w2, err := s.manager.Create(ctx, watcherID2)
	s.Require().NoError(err)
	s.Require().NoError(w2.SetHandler(s.mockHandler))
	s.startAndWait(ctx, w2)

	// Wait for events to be processed
	s.Require().Eventually(func() bool {
		return w1.Stats().LastProcessedSeqNum == 2 && w2.Stats().LastProcessedSeqNum == 2
	}, 200*time.Millisecond, 10*time.Millisecond)

	// Stop one watcher and ensure the other is still running
	w1.Stop(ctx)
	s.Require().Eventually(func() bool { return w1.Stats().State == watcher.StateStopped }, time.Second, 10*time.Millisecond)
	s.Equal(watcher.StateRunning, w2.Stats().State)

	// Stop the manager and ensure the second watcher is stopped
	err = s.manager.Stop(ctx)
	s.Require().NoError(err)
	s.Require().Equal(watcher.StateStopped, w1.Stats().State)
	s.Require().Equal(watcher.StateStopped, w2.Stats().State)

}

func (s *ManagerTestSuite) TestStoppingWatcherMultipleTimes() {
	ctx := context.Background()

	err := s.manager.Stop(ctx)
	s.Require().NoError(err)

	// Stopping an already stopped manager should not cause issues
	err = s.manager.Stop(ctx)
	s.Require().NoError(err)
}

func (s *ManagerTestSuite) startAndWait(ctx context.Context, w watcher.Watcher) {
	s.Require().NoError(w.Start(ctx))

	// Ensure the watcher is running
	s.Require().Eventually(func() bool {
		return w.Stats().State == watcher.StateRunning
	}, time.Second, 10*time.Millisecond)
}

func TestManagerSuite(t *testing.T) {
	suite.Run(t, new(ManagerTestSuite))
}
