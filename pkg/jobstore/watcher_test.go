//go:build unit || !integration

package jobstore

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type WatcherTestSuite struct {
	suite.Suite
	ctx context.Context
}

func TestWatcherTestSuite(t *testing.T) {
	suite.Run(t, new(WatcherTestSuite))
}

func (s *WatcherTestSuite) SetupTest() {
	s.ctx = context.Background()
}

func (s *WatcherTestSuite) TestCreateWatcher() {
	testcases := []struct {
		name                   string
		types                  StoreWatcherType
		events                 StoreEventType
		expected_watching_job  bool
		expected_watching_exec bool
		expected_watching_eval bool
		expected_create        bool
		expected_update        bool
		expected_delete        bool
	}{
		{
			name:                   "All the things",
			types:                  JobWatcher | ExecutionWatcher | EvaluationWatcher,
			events:                 CreateEvent | UpdateEvent | DeleteEvent,
			expected_watching_job:  true,
			expected_watching_exec: true,
			expected_watching_eval: true,
			expected_create:        true,
			expected_update:        true,
			expected_delete:        true,
		},
		{
			name:                   "Job creation events",
			types:                  JobWatcher,
			events:                 CreateEvent,
			expected_watching_eval: false,
			expected_watching_job:  true,
			expected_watching_exec: false,
			expected_create:        true,
			expected_update:        false,
			expected_delete:        false,
		},
		{
			name:                   "Job and Execution creation events",
			types:                  JobWatcher | ExecutionWatcher,
			events:                 CreateEvent,
			expected_watching_job:  true,
			expected_watching_exec: true,
			expected_watching_eval: false,
			expected_create:        true,
			expected_update:        false,
			expected_delete:        false,
		},
		{
			name:                   "Execution deletion events",
			types:                  ExecutionWatcher,
			events:                 DeleteEvent,
			expected_watching_job:  false,
			expected_watching_exec: true,
			expected_watching_eval: false,
			expected_create:        false,
			expected_update:        false,
			expected_delete:        true,
		},
		{
			name:                   "Updates for job and executions",
			types:                  ExecutionWatcher | JobWatcher,
			events:                 UpdateEvent,
			expected_watching_job:  true,
			expected_watching_exec: true,
			expected_watching_eval: false,
			expected_create:        false,
			expected_update:        true,
			expected_delete:        false,
		},
	}

	for _, tc := range testcases {
		s.Run(tc.name, func() {
			w := NewWatcher(context.Background(), tc.types, tc.events)

			watchingJob := w.IsWatchingType(JobWatcher)
			watchingExec := w.IsWatchingType(ExecutionWatcher)
			watchingEval := w.IsWatchingType(EvaluationWatcher)
			s.Equal(tc.expected_watching_job, watchingJob, "expectation around watching job not met")
			s.Equal(tc.expected_watching_exec, watchingExec, "expectation around watching exec not met")
			s.Equal(tc.expected_watching_eval, watchingEval, "expectation around watching evaluation not met")

			watchingCreate := w.IsWatchingEvent(CreateEvent)
			watchingUpdate := w.IsWatchingEvent(UpdateEvent)
			watchingDelete := w.IsWatchingEvent(DeleteEvent)
			s.Equal(tc.expected_create, watchingCreate, "expectation around watching create events not met ")
			s.Equal(tc.expected_update, watchingUpdate, "expectation around watching update events not met ")
			s.Equal(tc.expected_delete, watchingDelete, "expectation around watching delete  events not met ")

		})
	}
}

func (s *WatcherTestSuite) TestGetEvents() {
	testcases := []struct {
		name                string
		types               StoreWatcherType
		events              StoreEventType
		write_kind          StoreWatcherType
		write_event         StoreEventType
		expected_watchevent bool
		expected_object     any
	}{
		{
			name:                "all the things",
			types:               JobWatcher | ExecutionWatcher | EvaluationWatcher,
			events:              CreateEvent | UpdateEvent | DeleteEvent,
			write_kind:          JobWatcher,
			write_event:         CreateEvent,
			expected_watchevent: true,
			expected_object:     "all the things",
		},
		{
			name:                "create an evaluation",
			types:               JobWatcher | ExecutionWatcher | EvaluationWatcher,
			events:              CreateEvent | UpdateEvent | DeleteEvent,
			write_kind:          EvaluationWatcher,
			write_event:         CreateEvent,
			expected_watchevent: true,
			expected_object:     "create an evaluation",
		},
	}

	for _, tc := range testcases {
		s.Run(tc.name, func() {
			w := NewWatcher(context.Background(), tc.types, tc.events, WithFullChannelBehavior(WatcherBlock))
			ch := w.Channel()

			resp := w.write(tc.write_kind, tc.write_event, tc.name)
			s.Equal(tc.expected_watchevent, resp)

			msg := <-ch
			s.Equal(tc.types&msg.Kind, msg.Kind)
			s.Equal(tc.events&msg.Event, msg.Event)
			s.Equal(tc.write_kind, msg.Kind)
			s.Equal(tc.write_event, msg.Event)
			s.Equal(tc.expected_object, msg.Object, "msg was not as expected")
		})
	}
}

func (s *WatcherTestSuite) TestWriteEventWithFullChannel() {
	testcases := []struct {
		name                string
		fullChannelBehavior FullChannelBehavior
		firstEvent          any
		secondEvent         any
		expectedEvent       any
		expectedSecondResp  bool
		shouldBlock         bool
	}{
		{
			name:                "Block",
			fullChannelBehavior: WatcherBlock,
			firstEvent:          "first",
			secondEvent:         "second",
			expectedEvent:       "first", // The first event is not dropped
			expectedSecondResp:  true,
			shouldBlock:         true,
		},
		{
			name:                "Drop",
			fullChannelBehavior: WatcherDrop,
			firstEvent:          "first",
			secondEvent:         "second",
			expectedEvent:       "first", // The second event is dropped
			expectedSecondResp:  false,
		},
		{
			name:                "DropOldest",
			fullChannelBehavior: WatcherDropOldest,
			firstEvent:          "first",
			secondEvent:         "second",
			expectedEvent:       "second", // The first event is dropped
			expectedSecondResp:  true,
		},
	}

	for _, tc := range testcases {
		s.Run(tc.name, func() {
			w := NewWatcher(
				context.Background(),
				JobWatcher,
				CreateEvent,
				WithChannelSize(1),
				WithFullChannelBehavior(tc.fullChannelBehavior))

			wg := sync.WaitGroup{}
			wg.Add(1)

			go func() {
				w.write(JobWatcher, CreateEvent, tc.firstEvent)

				doneSecondEvent := make(chan bool)
				go func() {
					resp := w.write(JobWatcher, CreateEvent, tc.secondEvent)
					doneSecondEvent <- true
					s.Equal(tc.expectedSecondResp, resp, "resp was not as expected")
				}()

				timer := time.NewTimer(50 * time.Millisecond)
				select {
				case <-doneSecondEvent:
					timer.Stop()
					if tc.fullChannelBehavior == WatcherBlock {
						s.Failf("WriteEvent did not block when it should have", "fullChannelBehavior: %v", tc.fullChannelBehavior)
					}
				case <-timer.C:
					if tc.fullChannelBehavior != WatcherBlock {
						s.Failf("WriteEvent blocked when it should not have", "fullChannelBehavior: %v", tc.fullChannelBehavior)
					}
				}
				wg.Done()
			}()

			wg.Wait()
			msg := <-w.Channel()
			s.Equal(tc.expectedEvent, msg.Object, "msg was not as expected")
		})
	}
}

func (s *WatcherTestSuite) TestCloseWhenContextDone() {
	// Create a context that we can cancel
	ctx, cancel := context.WithCancel(s.ctx)

	// Create a watcher with this context
	w := NewWatcher(ctx, JobWatcher, CreateEvent)

	// Ensure the watcher is not closed to start with
	s.Require().False(w.closed, "Watcher should not be closed at start")

	// Cancel the context
	cancel()

	// Now the watcher should be closed
	s.Eventually(func() bool {
		return w.closed
	}, 100*time.Millisecond, 10*time.Millisecond, "Watcher should be closed after context is cancelled")
}

type WatchersManagerTestSuite struct {
	suite.Suite
	ctx context.Context
}

func TestWatchersManagerTestSuite(t *testing.T) {
	suite.Run(t, new(WatchersManagerTestSuite))
}

func (s *WatchersManagerTestSuite) SetupTest() {
	s.ctx = context.Background()
}

func (s *WatchersManagerTestSuite) TestNewWatcher() {
	manager := NewWatchersManager()
	watcher := manager.NewWatcher(s.ctx, JobWatcher, CreateEvent)

	s.NotNil(watcher, "Watcher should not be nil")
	s.True(watcher.IsWatchingType(JobWatcher), "Watcher should be watching JobWatcher type")
	s.True(watcher.IsWatchingEvent(CreateEvent), "Watcher should be watching CreateEvent")
}

func (s *WatchersManagerTestSuite) TestWrite() {
	manager := NewWatchersManager()
	watcher := manager.NewWatcher(s.ctx, JobWatcher, CreateEvent)

	manager.Write(JobWatcher, CreateEvent, "Test object")
	verifyWatcher(s.T(), watcher, JobWatcher, CreateEvent, "Test object")
}

func (s *WatchersManagerTestSuite) TestClose() {
	manager := NewWatchersManager()
	watcher := manager.NewWatcher(s.ctx, JobWatcher, CreateEvent)

	manager.Close()
	s.True(watcher.closed, "Watcher should be closed")
}

func (s *WatchersManagerTestSuite) TestWriteToMultipleWatchers() {
	manager := NewWatchersManager()

	// Create three watchers
	watcher1 := manager.NewWatcher(s.ctx, JobWatcher, CreateEvent)
	watcher2 := manager.NewWatcher(s.ctx, JobWatcher, CreateEvent)
	watcher3 := manager.NewWatcher(s.ctx, ExecutionWatcher, UpdateEvent)
	watcher4 := manager.NewWatcher(s.ctx, EvaluationWatcher, DeleteEvent)

	// Write an event to JobWatcher with CreateEvent
	manager.Write(JobWatcher, CreateEvent, "Test object 1")

	// Write an event to ExecutionWatcher with UpdateEvent
	manager.Write(ExecutionWatcher, UpdateEvent, "Test object 2")

	// Verify watcher1 and watcher2 received "Test object 1" and nothing else
	verifyWatcher(s.T(), watcher1, JobWatcher, CreateEvent, "Test object 1")
	verifyWatcher(s.T(), watcher2, JobWatcher, CreateEvent, "Test object 1")

	// Verify watcher3 received "Test object 2" and nothing else
	verifyWatcher(s.T(), watcher3, ExecutionWatcher, UpdateEvent, "Test object 2")

	// Verify watcher4 received nothing
	verifyWatcher(s.T(), watcher4, EvaluationWatcher, DeleteEvent, "")
}

// verifyWatcher is a helper function to verify a watcher received a specific event
func verifyWatcher(t *testing.T, watcher *Watcher, kind StoreWatcherType, event StoreEventType, expectedObject any) {
	select {
	case receivedEvent := <-watcher.Channel():
		if expectedObject == "" {
			require.Fail(t, "Watcher received an event when it should not have")
		} else {
			require.Equal(t, kind, receivedEvent.Kind, "Event kind should match")
			require.Equal(t, event, receivedEvent.Event, "Event type should match")
			require.Equal(t, expectedObject, receivedEvent.Object, "Event object should match")
		}
	case <-time.After(50 * time.Millisecond):
		if expectedObject != "" {
			require.Fail(t, "Timeout waiting for event")
		}
	}
}
