//go:build unit || !integration

package watcher_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/bacalhau-project/bacalhau/pkg/lib/watcher"
	"github.com/bacalhau-project/bacalhau/pkg/lib/watcher/boltdb"
	watchertest "github.com/bacalhau-project/bacalhau/pkg/lib/watcher/test"
)

type RegistryTestSuite struct {
	suite.Suite
	ctrl        *gomock.Controller
	mockStore   *watchertest.EventStoreWrapper
	mockHandler *watcher.MockEventHandler
	registry    watcher.Registry
}

func (s *RegistryTestSuite) SetupTest() {
	boltdbEventStore, err := boltdb.NewEventStore(watchertest.CreateBoltDB(s.T()))
	s.Require().NoError(err)

	s.ctrl = gomock.NewController(s.T())
	s.mockStore = watchertest.NewEventStoreWrapper(boltdbEventStore)
	s.mockHandler = watcher.NewMockEventHandler(s.ctrl)
	s.registry = watcher.NewRegistry(s.mockStore)
}

func (s *RegistryTestSuite) TearDownTest() {
	s.ctrl.Finish()
}

func (s *RegistryTestSuite) TestWatch() {
	ctx := context.Background()
	watcherID := "test-watcher"

	l, err := s.registry.Watch(ctx, watcherID, s.mockHandler)
	s.Require().NoError(err)
	s.Require().NotNil(l)
	s.Equal(watcherID, l.ID())

	// Stop the watcher to prevent further GetEvents calls
	l.Stop(ctx)
}

func (s *RegistryTestSuite) TestWatchDuplicateWatcher() {
	ctx := context.Background()
	watcherID := "test-watcher"

	l, err := s.registry.Watch(ctx, watcherID, s.mockHandler)
	s.Require().NoError(err)
	defer l.Stop(ctx)

	_, err = s.registry.Watch(ctx, watcherID, s.mockHandler)
	s.Require().Error(err)
	s.Contains(err.Error(), "watcher already exists")
}

func (s *RegistryTestSuite) TestGetWatcher() {
	ctx := context.Background()
	watcherID := "test-watcher"

	l, err := s.registry.Watch(ctx, watcherID, s.mockHandler)
	s.Require().NoError(err)
	defer l.Stop(ctx)

	retrievedWatcher, err := s.registry.GetWatcher(watcherID)
	s.Require().NoError(err)
	s.Require().NotNil(retrievedWatcher)
	s.Equal(watcherID, retrievedWatcher.ID())
}

func (s *RegistryTestSuite) TestGetNonExistentWatcher() {
	_, err := s.registry.GetWatcher("non-existent")
	s.Require().Error(err)
	s.Contains(err.Error(), "watcher not found")
}

func (s *RegistryTestSuite) TestStop() {
	ctx := context.Background()
	watcherID := "test-watcher"

	l, err := s.registry.Watch(ctx, watcherID, s.mockHandler)
	s.Require().NoError(err)

	// Ensure the watcher is running
	s.Eventually(func() bool {
		return l.Stats().State == watcher.StateRunning
	}, 1*time.Second, 10*time.Millisecond)

	err = s.registry.Stop(ctx)
	s.Require().NoError(err)
	s.Require().Equal(watcher.StateStopped, l.Stats().State)
}

func (s *RegistryTestSuite) TestStopWithTimeout() {
	ctx := context.Background()
	watcherID := "test-watcher"

	// Create a channel to control GetEvents
	getEventsCh := make(chan struct{})

	//// Set up the mockStore to block on GetEvents
	s.mockStore.WithGetEventsInterceptor(func() error {
		<-getEventsCh
		return nil
	})

	l, err := s.registry.Watch(ctx, watcherID, s.mockHandler)
	s.Require().NoError(err)

	// Ensure the watcher is running
	s.Eventually(func() bool {
		return l.Stats().State == watcher.StateRunning
	}, 200*time.Millisecond, 10*time.Millisecond)

	// Create a very short timeout
	ctxWithTimeout, cancel := context.WithTimeout(ctx, 1*time.Nanosecond)
	defer cancel()

	err = s.registry.Stop(ctxWithTimeout)
	s.Require().Error(err)
	s.Equal(context.DeadlineExceeded, err)

	// Ensure the watcher is stopping
	s.Eventually(func() bool {
		return l.Stats().State == watcher.StateStopping
	}, 200*time.Millisecond, 10*time.Millisecond)

	// sleep and verify that the watcher is still stopping
	time.Sleep(100 * time.Millisecond)
	s.Require().Equal(watcher.StateStopping, l.Stats().State)

	// Unblock GetEvents
	close(getEventsCh)

	// Ensure the watcher is stopped
	s.Eventually(func() bool {
		return l.Stats().State == watcher.StateStopped
	}, 200*time.Millisecond, 10*time.Millisecond)
}

func (s *RegistryTestSuite) TestWatcherProcessesEvents() {
	ctx := context.Background()
	watcherID := "test-watcher"

	events := []watcher.Event{
		{SeqNum: 1, Operation: watcher.OperationCreate, ObjectType: "TestObject", Object: "test1"},
		{SeqNum: 2, Operation: watcher.OperationUpdate, ObjectType: "TestObject", Object: "test2"},
	}

	for _, event := range events {
		err := s.mockStore.StoreEvent(ctx, event.Operation, event.ObjectType, event.Object)
		s.Require().NoError(err)
	}

	s.mockHandler.EXPECT().HandleEvent(gomock.Any(), gomock.Any()).Return(nil).Times(2)

	_, err := s.registry.Watch(ctx, watcherID, s.mockHandler)
	s.Require().NoError(err)

	// Wait for events to be processed
	time.Sleep(100 * time.Millisecond)

	err = s.registry.Stop(ctx)
	s.Require().NoError(err)
}

func (s *RegistryTestSuite) TestMultipleWatchers() {
	ctx := context.Background()
	watcherID1 := "test-watcher-1"
	watcherID2 := "test-watcher-2"

	events := []watcher.Event{
		{SeqNum: 1, Operation: watcher.OperationCreate, ObjectType: "TestObject", Object: "test1"},
		{SeqNum: 2, Operation: watcher.OperationUpdate, ObjectType: "TestObject", Object: "test2"},
	}

	for _, event := range events {
		err := s.mockStore.StoreEvent(ctx, event.Operation, event.ObjectType, event.Object)
		s.Require().NoError(err)
	}

	s.mockHandler.EXPECT().HandleEvent(gomock.Any(), gomock.Any()).Return(nil).Times(4)

	l1, err := s.registry.Watch(ctx, watcherID1, s.mockHandler)
	s.Require().NoError(err)
	l2, err := s.registry.Watch(ctx, watcherID2, s.mockHandler)
	s.Require().NoError(err)

	time.Sleep(100 * time.Millisecond)

	// Stop one watcher and ensure the other is still running
	l1.Stop(ctx)
	s.Eventually(func() bool { return l1.Stats().State == watcher.StateStopped }, time.Second, 10*time.Millisecond)
	s.Equal(watcher.StateRunning, l2.Stats().State)

	l2.Stop(ctx)
}

func (s *RegistryTestSuite) TestEventStoreErrors() {
	ctx := context.Background()
	watcherID := "test-watcher"

	// Test GetCheckpoint error
	s.mockStore.WithGetCheckpointInterceptor(func() error {
		return errors.New("checkpoint error")
	})
	_, err := s.registry.Watch(ctx, watcherID, s.mockHandler)
	s.Require().Error(err)
	s.Contains(err.Error(), "checkpoint error")

	// Reset checkpoint error
	s.mockStore.WithGetCheckpointInterceptor(nil)

	// Test GetEvents error
	s.mockStore.WithGetEventsInterceptor(func() error {
		return errors.New("et events error")
	})

	l, err := s.registry.Watch(ctx, watcherID, s.mockHandler)
	s.Require().NoError(err)
	time.Sleep(100 * time.Millisecond)
	s.Equal(watcher.StateRunning, l.Stats().State) // The watcher should keep running despite errors
}

func (s *RegistryTestSuite) TestRestartStoppedWatcher() {
	ctx := context.Background()
	watcherID := "test-watcher"

	l, err := s.registry.Watch(ctx, watcherID, s.mockHandler)
	s.Require().NoError(err)

	// Wait for the watcher to start
	s.Eventually(func() bool { return l.Stats().State == watcher.StateRunning }, 200*time.Millisecond, 10*time.Millisecond)

	l.Stop(ctx)
	s.Eventually(func() bool { return l.Stats().State == watcher.StateStopped }, 200*time.Second, 10*time.Millisecond)

	// Try to create a new watcher with the same ID
	newL, err := s.registry.Watch(ctx, watcherID, s.mockHandler)
	s.Require().NoError(err)
	s.NotEqual(l, newL)
}

func (s *RegistryTestSuite) TestStoppingWatcherMultipleTimes() {
	ctx := context.Background()

	err := s.registry.Stop(ctx)
	s.Require().NoError(err)

	// Stopping an already stopped registry should not cause issues
	err = s.registry.Stop(ctx)
	s.Require().NoError(err)
}

func TestRegistrySuite(t *testing.T) {
	suite.Run(t, new(RegistryTestSuite))
}
