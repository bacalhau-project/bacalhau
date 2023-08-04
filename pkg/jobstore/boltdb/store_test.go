//go:build unit || !integration

package boltjobstore

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/benbjohnson/clock"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type BoltJobstoreTestSuite struct {
	suite.Suite
	store  *BoltJobStore
	dbFile string
	ctx    context.Context
	clock  *clock.Mock
}

func TestBoltJobstoreTestSuite(t *testing.T) {
	suite.Run(t, new(BoltJobstoreTestSuite))
}

func (s *BoltJobstoreTestSuite) SetupTest() {
	s.clock = clock.NewMock()

	dir, _ := os.MkdirTemp("", "bacalhau-executionstore")
	s.dbFile = filepath.Join(dir, "test.boltdb")

	s.store, _ = NewBoltJobStore(s.dbFile, WithClock(s.clock))
	s.ctx = context.Background()

	jobFixtures := []struct {
		id              string
		client          string
		tags            []string
		jobStates       []model.JobStateType
		executionStates []model.ExecutionStateType
	}{
		{
			id:              "1",
			client:          "client1",
			tags:            []string{"gpu", "fast"},
			jobStates:       []model.JobStateType{model.JobStateQueued, model.JobStateInProgress, model.JobStateCancelled},
			executionStates: []model.ExecutionStateType{model.ExecutionStateAskForBid, model.ExecutionStateAskForBidAccepted, model.ExecutionStateCancelled},
		},
		{
			id:              "2",
			client:          "client2",
			tags:            []string{},
			jobStates:       []model.JobStateType{model.JobStateQueued, model.JobStateInProgress, model.JobStateCancelled},
			executionStates: []model.ExecutionStateType{model.ExecutionStateAskForBid, model.ExecutionStateAskForBidAccepted, model.ExecutionStateCancelled},
		},
		{
			id:              "3",
			client:          "client3",
			tags:            []string{"slow"},
			jobStates:       []model.JobStateType{model.JobStateQueued, model.JobStateInProgress},
			executionStates: []model.ExecutionStateType{model.ExecutionStateAskForBid, model.ExecutionStateAskForBidAccepted},
		},
	}

	for _, fixture := range jobFixtures {
		s.clock.Add(1 * time.Second)
		job := makeJob(
			model.EngineDocker,
			model.PublisherNoop,
			[]string{"bash", "-c", "echo hello"})
		job.Spec.Annotations = fixture.tags
		job.Metadata.ID = fixture.id
		job.Metadata.ClientID = fixture.client
		err := s.store.CreateJob(s.ctx, *job)
		s.NoError(err)

		s.clock.Add(1 * time.Second)
		execution := model.ExecutionState{
			JobID:            fixture.id,
			NodeID:           "nodeid",
			ComputeReference: "e-computeRef",
			State:            model.ExecutionStateNew,
		}
		err = s.store.CreateExecution(s.ctx, execution)
		s.NoError(err)

		for i, state := range fixture.jobStates {
			s.clock.Add(1 * time.Second)

			oldState := model.JobStateNew
			if i > 0 {
				oldState = fixture.jobStates[i-1]
			}

			request := jobstore.UpdateJobStateRequest{
				JobID:    fixture.id,
				NewState: state,
				Condition: jobstore.UpdateJobCondition{
					ExpectedState:   oldState,
					ExpectedVersion: i + 1,
				},
				Comment: fmt.Sprintf("moved to %+v", state),
			}
			err = s.store.UpdateJobState(s.ctx, request)
			s.NoError(err)
		}

		for i, state := range fixture.executionStates {
			s.clock.Add(1 * time.Second)

			oldState := model.ExecutionStateNew
			if i > 0 {
				oldState = fixture.executionStates[i-1]
			}

			// We are pretending this is a new execution struct
			execution.State = state
			execution.UpdateTime = s.clock.Now()

			request := jobstore.UpdateExecutionRequest{
				ExecutionID: execution.ID(),
				Condition: jobstore.UpdateExecutionCondition{
					ExpectedStates:  []model.ExecutionStateType{oldState},
					ExpectedVersion: i + 1,
				},
				NewValues: execution,
				Comment:   fmt.Sprintf("exec update to %+v", state),
			}

			err = s.store.UpdateExecution(s.ctx, request)
			s.NoError(err)
		}

	}
}

func (s *BoltJobstoreTestSuite) TearDownTest() {
	s.store.Close(s.ctx)
	os.Remove(s.dbFile)
}

func (s *BoltJobstoreTestSuite) TestUnfilteredJobHistory() {
	history, err := s.store.GetJobHistory(s.ctx, "1", jobstore.JobHistoryFilterOptions{})
	require.NoError(s.T(), err, "failed to get job history")
	require.Equal(s.T(), 8, len(history))
}

func (s *BoltJobstoreTestSuite) TestJobHistoryOrdering() {
	history, err := s.store.GetJobHistory(s.ctx, "1", jobstore.JobHistoryFilterOptions{})
	require.NoError(s.T(), err, "failed to get job history")

	// There are 6 history entries that we created directly, and 2 created by
	// CreateJob and CreateExecution
	require.Equal(s.T(), 8, len(history))

	// Make sure they come back in order
	values := make([]int64, len(history))
	for i, h := range history {
		values[i] = h.Time.Unix()
	}

	require.Equal(s.T(), []int64{1, 2, 3, 4, 5, 6, 7, 8}, values)
}

func (s *BoltJobstoreTestSuite) TestTimeFilteredJobHistory() {
	options := jobstore.JobHistoryFilterOptions{
		Since: 5,
	}

	history, err := s.store.GetJobHistory(s.ctx, "1", options)
	require.NoError(s.T(), err, "failed to get job history")
	require.Equal(s.T(), 4, len(history))
}

func (s *BoltJobstoreTestSuite) TestLevelFilteredJobHistory() {
	jobOptions := jobstore.JobHistoryFilterOptions{
		ExcludeExecutionLevel: true,
	}
	execOptions := jobstore.JobHistoryFilterOptions{
		ExcludeJobLevel: true,
	}

	history, err := s.store.GetJobHistory(s.ctx, "1", jobOptions)
	s.NoError(err, "failed to get job history")
	s.Equal(4, len(history))
	s.Equal(model.JobStateQueued, history[1].JobState.New)

	count := lo.Reduce(history, func(agg int, item model.JobHistory, _ int) int {
		if item.Type == model.JobHistoryTypeJobLevel {
			return agg + 1
		}
		return agg
	}, 0)
	s.Equal(count, 4)

	history, err = s.store.GetJobHistory(s.ctx, "1", execOptions)
	s.NoError(err, "failed to get job history")
	s.Equal(4, len(history))
	s.Equal(model.ExecutionStateAskForBid, history[1].ExecutionState.New)

	count = lo.Reduce(history, func(agg int, item model.JobHistory, _ int) int {
		if item.Type == model.JobHistoryTypeExecutionLevel {
			return agg + 1
		}
		return agg
	}, 0)
	s.Equal(count, 4)
}

func (s *BoltJobstoreTestSuite) TestSearchJobs() {
	s.T().Run("by client ID", func(t *testing.T) {
		jobs, err := s.store.GetJobs(s.ctx, jobstore.JobQuery{
			ClientID: "client1",
		})
		require.NoError(t, err)
		require.Equal(t, 1, len(jobs))
	})

	s.T().Run("by client ID and included tags", func(t *testing.T) {
		jobs, err := s.store.GetJobs(s.ctx, jobstore.JobQuery{
			ClientID:    "client1",
			IncludeTags: []model.IncludedTag{"fast", "slow"},
		})
		require.NoError(t, err)
		require.Equal(t, 1, len(jobs))
		require.Equal(t, "client1", jobs[0].Metadata.ClientID)
		require.Contains(t, jobs[0].Spec.Annotations, "fast")
		require.NotContains(t, jobs[0].Spec.Annotations, "slow")
	})

	s.T().Run("everything sorted by id", func(t *testing.T) {
		jobs, err := s.store.GetJobs(s.ctx, jobstore.JobQuery{
			ReturnAll: true,
			SortBy:    "id",
		})
		require.NoError(t, err)
		require.Equal(t, 3, len(jobs))
		ids := lo.Map(jobs, func(item model.Job, _ int) string {
			return item.ID()
		})
		require.EqualValues(t, []string{"1", "2", "3"}, ids)

		jobs, err = s.store.GetJobs(s.ctx, jobstore.JobQuery{
			ReturnAll:   true,
			SortBy:      "id",
			SortReverse: true,
		})
		require.NoError(t, err)
		require.Equal(t, 3, len(jobs))

		ids = lo.Map(jobs, func(item model.Job, _ int) string {
			return item.ID()
		})
		require.EqualValues(t, []string{"3", "2", "1"}, ids)
	})

	s.T().Run("everything", func(t *testing.T) {
		jobs, err := s.store.GetJobs(s.ctx, jobstore.JobQuery{
			ReturnAll: true,
		})
		require.NoError(t, err)
		require.Equal(t, 3, len(jobs))
	})

	s.T().Run("everything offset", func(t *testing.T) {
		jobs, err := s.store.GetJobs(s.ctx, jobstore.JobQuery{
			ReturnAll: true,
			Offset:    1,
		})
		require.NoError(t, err)
		require.Equal(t, 2, len(jobs))
	})

	s.T().Run("everything limit", func(t *testing.T) {
		jobs, err := s.store.GetJobs(s.ctx, jobstore.JobQuery{
			ReturnAll: true,
			Limit:     2,
		})
		require.NoError(t, err)
		require.Equal(t, 2, len(jobs))
	})

	s.T().Run("everything offset/limit", func(t *testing.T) {
		jobs, err := s.store.GetJobs(s.ctx, jobstore.JobQuery{
			ReturnAll: true,
			Offset:    1,
			Limit:     1,
		})
		require.NoError(t, err)
		require.Equal(t, 1, len(jobs))
	})

	s.T().Run("include tags", func(t *testing.T) {
		jobs, err := s.store.GetJobs(s.ctx, jobstore.JobQuery{
			IncludeTags: []model.IncludedTag{"gpu"},
		})
		require.NoError(t, err)
		require.Equal(t, 1, len(jobs))
		require.Equal(t, "1", jobs[0].ID())
	})

	s.T().Run("all but exclude tags", func(t *testing.T) {
		jobs, err := s.store.GetJobs(s.ctx, jobstore.JobQuery{
			ReturnAll:   true,
			ExcludeTags: []model.ExcludedTag{"fast"},
		})
		require.NoError(t, err)
		require.Equal(t, 2, len(jobs))
	})

	s.T().Run("include/exclude same tag", func(t *testing.T) {
		jobs, err := s.store.GetJobs(s.ctx, jobstore.JobQuery{
			IncludeTags: []model.IncludedTag{"gpu"},
			ExcludeTags: []model.ExcludedTag{"fast"},
		})
		require.NoError(t, err)
		require.Equal(t, 0, len(jobs))
	})

	s.T().Run("by id", func(t *testing.T) {
		jobs, err := s.store.GetJobs(s.ctx, jobstore.JobQuery{
			ID: "1",
		})
		require.NoError(t, err)
		require.Equal(t, 1, len(jobs))
		require.Equal(t, "1", jobs[0].ID())
	})
}

func (s *BoltJobstoreTestSuite) TestDeleteJob() {
	job := makeJob(
		model.EngineDocker,
		model.PublisherNoop,
		[]string{"bash", "-c", "echo hello"})
	job.Spec.Annotations = []string{"tag"}
	job.Metadata.ID = "deleteme"
	job.Metadata.ClientID = "client1"

	err := s.store.CreateJob(s.ctx, *job)
	s.NoError(err)

	err = s.store.DeleteJob(s.ctx, job.Metadata.ID)
	s.NoError(err)
}

func (s *BoltJobstoreTestSuite) TestGetJob() {
	job, err := s.store.GetJob(s.ctx, "1")
	s.NoError(err)
	s.NotNil(job)

	_, err = s.store.GetJob(s.ctx, "100")
	s.Error(err)
}

func (s *BoltJobstoreTestSuite) TestGetJobState() {
	state, err := s.store.GetJobState(s.ctx, "1")
	s.NoError(err)
	s.NotNil(state)
	s.Greater(len(state.Executions), 0)

	_, err = s.store.GetJobState(s.ctx, "100")
	s.Error(err)
}

func (s *BoltJobstoreTestSuite) TestInProgressJobs() {
	infos, err := s.store.GetInProgressJobs(s.ctx)
	s.NoError(err)
	s.Equal(1, len(infos))
	s.Equal("3", infos[0].Job.ID())
}

func (s *BoltJobstoreTestSuite) TestEvents() {
	ch := s.store.Watch(s.ctx,
		jobstore.JobWatcher|jobstore.ExecutionWatcher,
		jobstore.CreateEvent|jobstore.UpdateEvent|jobstore.DeleteEvent,
	)

	job := makeJob(
		model.EngineDocker,
		model.PublisherNoop,
		[]string{"bash", "-c", "echo hello"})
	job.Metadata.ID = "10"
	job.Metadata.ClientID = "1"

	var execution model.ExecutionState

	s.Run("job create event", func() {
		err := s.store.CreateJob(s.ctx, *job)
		s.NoError(err)

		// Read an event, it should be a jobcreate
		ev := <-ch
		s.Equal(ev.Event, jobstore.CreateEvent)
		s.Equal(ev.Kind, jobstore.JobWatcher)

		var decodedJob model.Job
		err = json.Unmarshal(ev.Object, &decodedJob)
		s.NoError(err)
		s.Equal(decodedJob.ID(), job.ID())
	})

	s.Run("execution create event", func() {
		s.clock.Add(1 * time.Second)
		execution = model.ExecutionState{
			JobID:            "10",
			NodeID:           "nodeid",
			ComputeReference: "e-computeRef",
			State:            model.ExecutionStateNew,
		}
		err := s.store.CreateExecution(s.ctx, execution)
		s.NoError(err)

		// Read an event, it should be a Execution Create
		ev := <-ch
		s.Equal(ev.Event, jobstore.CreateEvent)
		s.Equal(ev.Kind, jobstore.ExecutionWatcher)
	})

	s.Run("update job state event", func() {
		request := jobstore.UpdateJobStateRequest{
			JobID:    "10",
			NewState: model.JobStateInProgress,
			Condition: jobstore.UpdateJobCondition{
				ExpectedState: model.JobStateNew,
			},
			Comment: "event test",
		}
		_ = s.store.UpdateJobState(s.ctx, request)
		ev := <-ch
		s.Equal(ev.Event, jobstore.UpdateEvent)
		s.Equal(ev.Kind, jobstore.JobWatcher)
	})

	s.Run("update execution state event", func() {
		execution.State = model.ExecutionStateAskForBid
		execution.UpdateTime = s.clock.Now()
		s.store.UpdateExecution(s.ctx, jobstore.UpdateExecutionRequest{
			ExecutionID: execution.ID(),
			Condition: jobstore.UpdateExecutionCondition{
				ExpectedStates: []model.ExecutionStateType{model.ExecutionStateNew},
			},
			NewValues: execution,
			Comment:   "event test",
		})
		ev := <-ch
		s.Equal(ev.Event, jobstore.UpdateEvent)
		s.Equal(ev.Kind, jobstore.ExecutionWatcher)

		var decodedExecution model.ExecutionState
		err := json.Unmarshal(ev.Object, &decodedExecution)
		s.NoError(err)
		s.Equal(decodedExecution.ID(), execution.ID())
	})

	s.Run("delete job event", func() {
		_ = s.store.DeleteJob(s.ctx, job.ID())
		ev := <-ch
		s.Equal(ev.Event, jobstore.DeleteEvent)
		s.Equal(ev.Kind, jobstore.JobWatcher)
	})
}

func (s *BoltJobstoreTestSuite) TestEvaluations() {

	eval := models.Evaluation{
		ID:    "e1",
		JobID: "10",
	}

	// Wrong job ID means JobNotFound
	err := s.store.CreateEvaluation(s.ctx, eval)
	s.Error(err)

	// Correct job ID
	eval.JobID = "1"
	err = s.store.CreateEvaluation(s.ctx, eval)
	s.NoError(err)

	_, err = s.store.GetEvaluation(s.ctx, "missing")
	s.Error(err)

	e, err := s.store.GetEvaluation(s.ctx, eval.ID)
	s.NoError(err)
	s.Equal(e, eval)

	err = s.store.DeleteEvaluation(s.ctx, eval.ID)
	s.NoError(err)
}

func makeJob(
	engineType model.Engine,
	publisherType model.Publisher,
	entrypointArray []string) *model.Job {
	j := model.NewJob()

	j.Spec = model.Spec{
		Engine: engineType,
		PublisherSpec: model.PublisherSpec{
			Type: publisherType,
		},
		Docker: model.JobSpecDocker{
			Image:      "ubuntu:latest",
			Entrypoint: entrypointArray,
		},
	}

	j.Spec.Deal = model.Deal{
		Concurrency: 1,
	}

	return j
}
