//go:build unit || !integration

package inmemory

import (
	"context"
	"fmt"
	"sort"
	"testing"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/test/mock"
	"github.com/google/uuid"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/benbjohnson/clock"

	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

type InMemoryTestSuite struct {
	suite.Suite
	store *InMemoryJobStore
	ctx   context.Context
	clock *clock.Mock
	ids   []string
}

func TestInMemoryTestSuite(t *testing.T) {
	suite.Run(t, new(InMemoryTestSuite))
}

func (s *InMemoryTestSuite) SetupTest() {
	s.clock = clock.NewMock()

	s.store = NewInMemoryJobStore(WithClock(s.clock))
	s.ctx = context.Background()

	jobFixtures := []struct {
		id              string
		client          string
		tags            map[string]string
		jobStates       []models.JobStateType
		executionStates []models.ExecutionStateType
	}{
		{
			id:              uuid.New().String(),
			client:          "client1",
			tags:            map[string]string{"gpu": "true", "fast": "true"},
			jobStates:       []models.JobStateType{models.JobStateTypePending, models.JobStateTypeRunning, models.JobStateTypeStopped},
			executionStates: []models.ExecutionStateType{models.ExecutionStateAskForBid, models.ExecutionStateAskForBidAccepted, models.ExecutionStateCancelled},
		},
		{
			id:              uuid.New().String(),
			client:          "client2",
			tags:            map[string]string{},
			jobStates:       []models.JobStateType{models.JobStateTypePending, models.JobStateTypeRunning, models.JobStateTypeStopped},
			executionStates: []models.ExecutionStateType{models.ExecutionStateAskForBid, models.ExecutionStateAskForBidAccepted, models.ExecutionStateCancelled},
		},
		{
			id:              uuid.New().String(),
			client:          "client3",
			tags:            map[string]string{"slow": "true"},
			jobStates:       []models.JobStateType{models.JobStateTypePending, models.JobStateTypeRunning},
			executionStates: []models.ExecutionStateType{models.ExecutionStateAskForBid, models.ExecutionStateAskForBidAccepted},
		},
	}

	for _, fixture := range jobFixtures {
		s.ids = append(s.ids, fixture.id)

		s.clock.Add(1 * time.Second)
		job := makeDockerEngineJob(
			[]string{"bash", "-c", "echo hello"})

		job.ID = fixture.id
		job.Labels = fixture.tags
		job.Namespace = fixture.client

		err := s.store.CreateJob(s.ctx, *job)
		s.NoError(err)

		s.clock.Add(1 * time.Second)

		execution := mock.ExecutionForJob(job)
		err = s.store.CreateExecution(s.ctx, *execution)
		s.NoError(err)

		for i, state := range fixture.jobStates {
			s.clock.Add(1 * time.Second)

			oldState := models.JobStateTypePending
			if i > 0 {
				oldState = fixture.jobStates[i-1]
			}

			request := jobstore.UpdateJobStateRequest{
				JobID:    fixture.id,
				NewState: state,
				Condition: jobstore.UpdateJobCondition{
					ExpectedState:    oldState,
					ExpectedRevision: uint64(i + 1),
				},
				Comment: fmt.Sprintf("moved to %+v", state),
			}
			err = s.store.UpdateJobState(s.ctx, request)
			s.NoError(err)
		}

		for i, state := range fixture.executionStates {
			s.clock.Add(1 * time.Second)

			oldState := models.ExecutionStateNew
			if i > 0 {
				oldState = fixture.executionStates[i-1]
			}

			// We are pretending this is a new execution struct
			execution.ComputeState.StateType = state
			execution.ModifyTime = s.clock.Now().UTC().UnixNano()

			request := jobstore.UpdateExecutionRequest{
				ExecutionID: execution.ID,
				Condition: jobstore.UpdateExecutionCondition{
					ExpectedStates:   []models.ExecutionStateType{oldState},
					ExpectedRevision: uint64(i + 1),
				},
				NewValues: *execution,
				Comment:   fmt.Sprintf("exec update to %+v", state),
			}

			err = s.store.UpdateExecution(s.ctx, request)
			s.NoError(err)
		}

	}
}

func (s *InMemoryTestSuite) TearDownTest() {
	s.store.Close(s.ctx)
	s.ids = []string{}
}

func (s *InMemoryTestSuite) TestUnfilteredJobHistory() {
	history, err := s.store.GetJobHistory(s.ctx, s.ids[0], jobstore.JobHistoryFilterOptions{})
	require.NoError(s.T(), err, "failed to get job history")
	require.Equal(s.T(), 8, len(history))
}

func (s *InMemoryTestSuite) TestJobHistoryOrdering() {
	history, err := s.store.GetJobHistory(s.ctx, s.ids[0], jobstore.JobHistoryFilterOptions{})
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

func (s *InMemoryTestSuite) TestTimeFilteredJobHistory() {
	options := jobstore.JobHistoryFilterOptions{
		Since: 5,
	}

	history, err := s.store.GetJobHistory(s.ctx, s.ids[0], options)
	require.NoError(s.T(), err, "failed to get job history")
	require.Equal(s.T(), 4, len(history))
}

func (s *InMemoryTestSuite) TestLevelFilteredJobHistory() {
	jobOptions := jobstore.JobHistoryFilterOptions{
		ExcludeExecutionLevel: true,
	}
	execOptions := jobstore.JobHistoryFilterOptions{
		ExcludeJobLevel: true,
	}

	history, err := s.store.GetJobHistory(s.ctx, s.ids[0], jobOptions)
	s.NoError(err, "failed to get job history")
	s.Equal(4, len(history))
	s.Equal(models.JobStateTypePending, history[1].JobState.New)

	count := lo.Reduce(history, func(agg int, item models.JobHistory, _ int) int {
		if item.Type == models.JobHistoryTypeJobLevel {
			return agg + 1
		}
		return agg
	}, 0)
	s.Equal(count, 4)

	history, err = s.store.GetJobHistory(s.ctx, s.ids[0], execOptions)
	s.NoError(err, "failed to get job history")
	s.Equal(4, len(history))
	s.Equal(models.ExecutionStateAskForBid, history[1].ExecutionState.New)

	count = lo.Reduce(history, func(agg int, item models.JobHistory, _ int) int {
		if item.Type == models.JobHistoryTypeExecutionLevel {
			return agg + 1
		}
		return agg
	}, 0)
	s.Equal(count, 4)
}

func (s *InMemoryTestSuite) TestSearchJobs() {
	s.T().Run("by client ID", func(t *testing.T) {
		jobs, err := s.store.GetJobs(s.ctx, jobstore.JobQuery{
			Namespace: "client1",
		})
		require.NoError(t, err)
		require.Equal(t, 1, len(jobs))
	})

	s.T().Run("by client ID and included tags", func(t *testing.T) {
		jobs, err := s.store.GetJobs(s.ctx, jobstore.JobQuery{
			Namespace:   "client1",
			IncludeTags: []string{"fast", "slow"},
		})
		require.NoError(t, err)
		require.Equal(t, 1, len(jobs))
		require.Equal(t, "client1", jobs[0].Namespace)
		require.Contains(t, jobs[0].Labels, "fast")
		require.NotContains(t, jobs[0].Labels, "slow")
	})

	s.T().Run("everything sorted by id", func(t *testing.T) {
		sorted_ids := append([]string(nil), s.ids...)
		reverse_sorted_ids := append([]string(nil), s.ids...)
		sort.Slice(sorted_ids, func(i, j int) bool {
			return sorted_ids[i] < sorted_ids[j]
		})
		sort.Slice(reverse_sorted_ids, func(i, j int) bool {
			return reverse_sorted_ids[i] > reverse_sorted_ids[j]
		})

		jobs, err := s.store.GetJobs(s.ctx, jobstore.JobQuery{
			ReturnAll: true,
			SortBy:    "id",
		})
		require.NoError(t, err)
		require.Equal(t, 3, len(jobs))

		ids := lo.Map(jobs, func(item models.Job, _ int) string {
			return item.ID
		})
		require.EqualValues(t, sorted_ids, ids)

		jobs, err = s.store.GetJobs(s.ctx, jobstore.JobQuery{
			ReturnAll:   true,
			SortBy:      "id",
			SortReverse: true,
		})
		require.NoError(t, err)
		require.Equal(t, 3, len(jobs))

		ids = lo.Map(jobs, func(item models.Job, _ int) string {
			return item.ID
		})

		require.EqualValues(t, reverse_sorted_ids, ids)
	})

	s.T().Run("everything", func(t *testing.T) {
		jobs, err := s.store.GetJobs(s.ctx, jobstore.JobQuery{
			ReturnAll: true,
		})
		require.NoError(t, err)
		require.Equal(t, 3, len(jobs))
	})

	s.T().Run("everything offset", func(t *testing.T) {
		// Offset ignored inmemorystore
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
		// Offset ignored inmemorystore
	})

	s.T().Run("include tags", func(t *testing.T) {
		jobs, err := s.store.GetJobs(s.ctx, jobstore.JobQuery{
			IncludeTags: []string{"gpu"},
		})
		require.NoError(t, err)
		require.Equal(t, 1, len(jobs))
		require.Equal(t, s.ids[0], jobs[0].ID)
	})

	s.T().Run("all but exclude tags", func(t *testing.T) {
		jobs, err := s.store.GetJobs(s.ctx, jobstore.JobQuery{
			ReturnAll:   true,
			ExcludeTags: []string{"fast"},
		})
		require.NoError(t, err)
		require.Equal(t, 2, len(jobs))
	})

	s.T().Run("include-exclude same tag", func(t *testing.T) {
		jobs, err := s.store.GetJobs(s.ctx, jobstore.JobQuery{
			IncludeTags: []string{"gpu"},
			ExcludeTags: []string{"fast"},
		})
		require.NoError(t, err)
		require.Equal(t, 0, len(jobs))
	})

	s.T().Run("by id", func(t *testing.T) {
		jobs, err := s.store.GetJobs(s.ctx, jobstore.JobQuery{
			ID: s.ids[0],
		})
		require.NoError(t, err)
		require.Equal(t, 1, len(jobs))
		require.Equal(t, s.ids[0], jobs[0].ID)
	})
}

func (s *InMemoryTestSuite) TestDeleteJob() {
	job := makeDockerEngineJob(
		[]string{"bash", "-c", "echo hello"})
	job.Labels = map[string]string{"tag": "value"}
	job.ID = "deleteme"
	job.Namespace = "client1"
	err := s.store.CreateJob(s.ctx, *job)
	s.NoError(err)

	err = s.store.DeleteJob(s.ctx, job.ID)
	s.NoError(err)
}

func (s *InMemoryTestSuite) TestGetJob() {
	job, err := s.store.GetJob(s.ctx, s.ids[0])
	s.NoError(err)
	s.NotNil(job)

	_, err = s.store.GetJob(s.ctx, "100")
	s.Error(err)
}

func (s *InMemoryTestSuite) TestGetExecutions() {
	id := s.ids[0]
	_, err := s.store.GetJob(s.ctx, id)
	s.NoError(err)

	state, err := s.store.GetExecutions(s.ctx, id)
	s.NoError(err)
	s.NotNil(state)
	s.Greater(len(state), 0)

	_, err = s.store.GetExecutions(s.ctx, uuid.New().String())
	s.Error(err)
}

func (s *InMemoryTestSuite) TestInProgressJobs() {
	jobs, err := s.store.GetInProgressJobs(s.ctx)
	s.NoError(err)
	s.Equal(1, len(jobs))
	last_id := s.ids[2]
	s.Equal(last_id, jobs[0].ID)
}

func (s *InMemoryTestSuite) TestEvents() {
	ch := s.store.Watch(s.ctx,
		jobstore.JobWatcher|jobstore.ExecutionWatcher,
		jobstore.CreateEvent|jobstore.UpdateEvent|jobstore.DeleteEvent,
	)

	job := makeDockerEngineJob(
		[]string{"bash", "-c", "echo hello"})
	job.ID = "10"
	job.Namespace = "1"

	var execution models.Execution

	s.Run("job create event", func() {
		err := s.store.CreateJob(s.ctx, *job)
		s.NoError(err)

		// Read an event, it should be a jobcreate
		ev := <-ch
		s.Equal(ev.Event, jobstore.CreateEvent)
		s.Equal(ev.Kind, jobstore.JobWatcher)

		decodedJob := ev.Object.(models.Job)
		s.Equal(decodedJob.ID, job.ID)
	})

	s.Run("execution create event", func() {
		s.clock.Add(1 * time.Second)
		execution = models.Execution{
			JobID:        "10",
			NodeID:       "nodeid",
			ID:           "e-computeRef",
			ComputeState: models.State[models.ExecutionStateType]{StateType: models.ExecutionStateNew},
		}
		err := s.store.CreateExecution(s.ctx, execution)
		s.NoError(err)

		// Read an event, it should be a ExecutionForJob Create
		ev := <-ch
		s.Equal(ev.Event, jobstore.CreateEvent)
		s.Equal(ev.Kind, jobstore.ExecutionWatcher)
	})

	s.Run("update job state event", func() {
		request := jobstore.UpdateJobStateRequest{
			JobID:    "10",
			NewState: models.JobStateTypeRunning,
			Condition: jobstore.UpdateJobCondition{
				ExpectedState: models.JobStateTypePending,
			},
			Comment: "event test",
		}
		_ = s.store.UpdateJobState(s.ctx, request)
		ev := <-ch
		s.Equal(ev.Event, jobstore.UpdateEvent)
		s.Equal(ev.Kind, jobstore.JobWatcher)
	})

	s.Run("update execution state event", func() {
		execution.ComputeState.StateType = models.ExecutionStateAskForBid
		execution.ModifyTime = s.clock.Now().UTC().UnixNano()
		s.store.UpdateExecution(s.ctx, jobstore.UpdateExecutionRequest{
			ExecutionID: execution.ID,
			Condition: jobstore.UpdateExecutionCondition{
				ExpectedStates: []models.ExecutionStateType{models.ExecutionStateNew},
			},
			NewValues: execution,
			Comment:   "event test",
		})
		ev := <-ch
		s.Equal(ev.Event, jobstore.UpdateEvent)
		s.Equal(ev.Kind, jobstore.ExecutionWatcher)

		decodedExecution := ev.Object.(models.Execution)
		s.Equal(decodedExecution.ID, execution.ID)
	})

	s.Run("delete job event", func() {
		_ = s.store.DeleteJob(s.ctx, job.ID)
		ev := <-ch
		s.Equal(ev.Event, jobstore.DeleteEvent)
		s.Equal(ev.Kind, jobstore.JobWatcher)
	})
}

func (s *InMemoryTestSuite) TestEvaluations() {

	eval := models.Evaluation{
		ID:    "e1",
		JobID: "10",
	}

	// Wrong job ID means JobNotFound
	err := s.store.CreateEvaluation(s.ctx, eval)
	s.Error(err)

	// Correct job ID
	eval.JobID = s.ids[0]
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

func makeDockerEngineJob(entrypointArray []string) *models.Job {
	j := mock.Job()
	j.Task().Engine = &models.SpecConfig{
		Type: models.EngineDocker,
		Params: map[string]interface{}{
			"Image":      "ubuntu:latest",
			"Entrypoint": entrypointArray,
		},
	}
	return j
}
