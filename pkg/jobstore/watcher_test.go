//go:build unit || !integration

package jobstore_test

import (
	"context"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
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
		types                  jobstore.StoreWatcherType
		events                 jobstore.StoreEventType
		expected_watching_job  bool
		expected_watching_exec bool
		expected_create        bool
		expected_update        bool
		expected_delete        bool
	}{
		{
			name:                   "All the things",
			types:                  jobstore.JobWatcher | jobstore.ExecutionWatcher,
			events:                 jobstore.CreateEvent | jobstore.UpdateEvent | jobstore.DeleteEvent,
			expected_watching_job:  true,
			expected_watching_exec: true,
			expected_create:        true,
			expected_update:        true,
			expected_delete:        true,
		},
		{
			name:                   "Job creation events",
			types:                  jobstore.JobWatcher,
			events:                 jobstore.CreateEvent,
			expected_watching_job:  true,
			expected_watching_exec: false,
			expected_create:        true,
			expected_update:        false,
			expected_delete:        false,
		},
		{
			name:                   "Job and Execution creation events",
			types:                  jobstore.JobWatcher | jobstore.ExecutionWatcher,
			events:                 jobstore.CreateEvent,
			expected_watching_job:  true,
			expected_watching_exec: true,
			expected_create:        true,
			expected_update:        false,
			expected_delete:        false,
		},
		{
			name:                   "Execution deletion events",
			types:                  jobstore.ExecutionWatcher,
			events:                 jobstore.DeleteEvent,
			expected_watching_job:  false,
			expected_watching_exec: true,
			expected_create:        false,
			expected_update:        false,
			expected_delete:        true,
		},
		{
			name:                   "Updates for job and executions",
			types:                  jobstore.ExecutionWatcher | jobstore.JobWatcher,
			events:                 jobstore.UpdateEvent,
			expected_watching_job:  true,
			expected_watching_exec: true,
			expected_create:        false,
			expected_update:        true,
			expected_delete:        false,
		},
	}

	for _, tc := range testcases {
		s.Run(tc.name, func() {
			w := jobstore.NewWatcher(tc.types, tc.events)

			watchingJob := w.IsWatchingType(jobstore.JobWatcher)
			watchingExec := w.IsWatchingType(jobstore.ExecutionWatcher)
			s.Equal(tc.expected_watching_job, watchingJob, "expectation around watching job not met")
			s.Equal(tc.expected_watching_exec, watchingExec, "expectation around watching exec not met")

			watchingCreate := w.IsWatchingEvent(jobstore.CreateEvent)
			watchingUpdate := w.IsWatchingEvent(jobstore.UpdateEvent)
			watchingDelete := w.IsWatchingEvent(jobstore.DeleteEvent)
			s.Equal(tc.expected_create, watchingCreate, "expectation around watching create events not met ")
			s.Equal(tc.expected_update, watchingUpdate, "expectation around watching update events not met ")
			s.Equal(tc.expected_delete, watchingDelete, "expectation around watching delete  events not met ")

		})
	}
}

func (s *WatcherTestSuite) TestGetEvents() {
	testcases := []struct {
		name                string
		types               jobstore.StoreWatcherType
		events              jobstore.StoreEventType
		write_kind          jobstore.StoreWatcherType
		write_event         jobstore.StoreEventType
		expected_watchevent bool
		expected_object     []byte
	}{
		{
			name:                "all the things",
			types:               jobstore.JobWatcher | jobstore.ExecutionWatcher,
			events:              jobstore.CreateEvent | jobstore.UpdateEvent | jobstore.DeleteEvent,
			write_kind:          jobstore.JobWatcher,
			write_event:         jobstore.CreateEvent,
			expected_watchevent: true,
			expected_object:     []byte("all the things"),
		},
	}

	for _, tc := range testcases {
		s.Run(tc.name, func() {
			w := jobstore.NewWatcher(tc.types, tc.events)
			ch := w.Channel()

			resp := w.WriteEvent(tc.write_kind, tc.write_event, []byte(tc.name), true)
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
