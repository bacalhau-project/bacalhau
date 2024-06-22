//go:build unit || !integration

package boltjobstore

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/benbjohnson/clock"
	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"k8s.io/apimachinery/pkg/labels"

	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/test/mock"
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
		jobType         string
		client          string
		tags            map[string]string
		jobStates       []models.JobStateType
		executionStates []models.ExecutionStateType
	}{
		{
			id:              "110",
			client:          "client1",
			jobType:         "batch",
			tags:            map[string]string{"gpu": "true", "fast": "true"},
			jobStates:       []models.JobStateType{models.JobStateTypePending, models.JobStateTypeRunning, models.JobStateTypeStopped},
			executionStates: []models.ExecutionStateType{models.ExecutionStateAskForBid, models.ExecutionStateAskForBidAccepted, models.ExecutionStateCancelled},
		},
		{
			id:              "120",
			client:          "client2",
			jobType:         "batch",
			tags:            map[string]string{},
			jobStates:       []models.JobStateType{models.JobStateTypePending, models.JobStateTypeRunning, models.JobStateTypeStopped},
			executionStates: []models.ExecutionStateType{models.ExecutionStateAskForBid, models.ExecutionStateAskForBidAccepted, models.ExecutionStateCancelled},
		},
		{
			id:              "130",
			client:          "client3",
			jobType:         "batch",
			tags:            map[string]string{"slow": "true", "max": "10"},
			jobStates:       []models.JobStateType{models.JobStateTypePending, models.JobStateTypeRunning},
			executionStates: []models.ExecutionStateType{models.ExecutionStateAskForBid, models.ExecutionStateAskForBidAccepted},
		},
		{
			id:              "140",
			client:          "client4",
			jobType:         "batch",
			tags:            map[string]string{"max": "10"},
			jobStates:       []models.JobStateType{models.JobStateTypePending, models.JobStateTypeRunning},
			executionStates: []models.ExecutionStateType{models.ExecutionStateAskForBid, models.ExecutionStateAskForBidAccepted},
		},
		{
			id:              "150",
			client:          "client5",
			jobType:         "daemon",
			tags:            map[string]string{"max": "10"},
			jobStates:       []models.JobStateType{models.JobStateTypePending, models.JobStateTypeRunning},
			executionStates: []models.ExecutionStateType{models.ExecutionStateAskForBid, models.ExecutionStateAskForBidAccepted},
		},
	}

	for _, fixture := range jobFixtures {
		s.clock.Add(1 * time.Second)
		job := makeDockerEngineJob(
			[]string{"bash", "-c", "echo hello"})

		job.ID = fixture.id
		job.Type = fixture.jobType
		job.Labels = fixture.tags
		job.Namespace = fixture.client
		s.Require().NoError(s.store.CreateJob(s.ctx, *job))
		s.Require().NoError(s.store.AddJobHistory(s.ctx, fixture.id, *models.NewEvent("test").WithMessage("job created")))

		s.clock.Add(1 * time.Second)
		execution := mock.ExecutionForJob(job)
		execution.ComputeState.StateType = models.ExecutionStateNew
		// clear out CreateTime and ModifyTime from the mocked execution to let the job store fill those
		execution.CreateTime = 0
		execution.ModifyTime = 0
		s.Require().NoError(s.store.CreateExecution(s.ctx, *execution))
		s.Require().NoError(s.store.AddExecutionHistory(s.ctx, fixture.id, execution.ID, *models.NewEvent("test").WithMessage("execution created")))

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
			}
			s.Require().NoError(s.store.UpdateJobState(s.ctx, request))
			s.Require().NoError(s.store.AddJobHistory(s.ctx, fixture.id, *models.NewEvent("test").WithMessage(state.String())))
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
			}

			s.Require().NoError(s.store.UpdateExecution(s.ctx, request))
			s.Require().NoError(s.store.AddExecutionHistory(s.ctx, fixture.id, execution.ID, *models.NewEvent("test").WithMessage(state.String())))
		}

	}
}

func (s *BoltJobstoreTestSuite) TearDownTest() {
	s.store.Close(s.ctx)
	os.Remove(s.dbFile)
}

func (s *BoltJobstoreTestSuite) TestUnfilteredJobHistory() {
	history, err := s.store.GetJobHistory(s.ctx, "110", jobstore.JobHistoryFilterOptions{})
	s.Require().NoError(err, "failed to get job history")
	s.Require().Equal(8, len(history))

	history, err = s.store.GetJobHistory(s.ctx, "11", jobstore.JobHistoryFilterOptions{})
	s.Require().NoError(err)
	s.NotEmpty(history)
	s.Require().Equal("110", history[0].JobID)

	history, err = s.store.GetJobHistory(s.ctx, "1", jobstore.JobHistoryFilterOptions{})
	s.Require().Error(err)
	s.Require().IsType(err, &bacerrors.MultipleJobsFound{})
	s.Require().Nil(history)
}

func (s *BoltJobstoreTestSuite) TestJobHistoryOrdering() {
	history, err := s.store.GetJobHistory(s.ctx, "110", jobstore.JobHistoryFilterOptions{})
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

	history, err := s.store.GetJobHistory(s.ctx, "110", options)
	require.NoError(s.T(), err, "failed to get job history")
	require.Equal(s.T(), 4, len(history))
}

func (s *BoltJobstoreTestSuite) TestExecutionFilteredJobHistory() {
	allHistories, err := s.store.GetJobHistory(s.ctx, "110", jobstore.JobHistoryFilterOptions{})
	require.NoError(s.T(), err)

	var executionID string
	for _, h := range allHistories {
		if h.ExecutionID != "" {
			executionID = h.ExecutionID
			break
		}
	}
	require.NotEmpty(s.T(), executionID, "failed to find execution ID")

	options := jobstore.JobHistoryFilterOptions{
		ExecutionID: executionID,
	}

	history, err := s.store.GetJobHistory(s.ctx, "110", options)
	require.NoError(s.T(), err, "failed to get job history")

	for _, h := range history {
		require.Equal(s.T(), executionID, h.ExecutionID)
	}
}

func (s *BoltJobstoreTestSuite) TestLevelFilteredJobHistory() {
	jobOptions := jobstore.JobHistoryFilterOptions{
		ExcludeExecutionLevel: true,
	}
	execOptions := jobstore.JobHistoryFilterOptions{
		ExcludeJobLevel: true,
	}

	history, err := s.store.GetJobHistory(s.ctx, "110", jobOptions)
	s.Require().NoError(err, "failed to get job history")
	s.Require().Equal(4, len(history))

	count := lo.Reduce(history, func(agg int, item models.JobHistory, _ int) int {
		if item.Type == models.JobHistoryTypeJobLevel {
			return agg + 1
		}
		return agg
	}, 0)
	s.Require().Equal(count, 4)

	history, err = s.store.GetJobHistory(s.ctx, "110", execOptions)
	s.Require().NoError(err, "failed to get job history")
	s.Require().Equal(4, len(history))

	count = lo.Reduce(history, func(agg int, item models.JobHistory, _ int) int {
		if item.Type == models.JobHistoryTypeExecutionLevel {
			return agg + 1
		}
		return agg
	}, 0)
	s.Require().Equal(count, 4)
}

func (s *BoltJobstoreTestSuite) TestSearchJobs() {
	s.T().Run("by client ID and included tags", func(t *testing.T) {
		response, err := s.store.GetJobs(s.ctx, jobstore.JobQuery{
			Namespace:   "client1",
			IncludeTags: []string{"fast", "slow"},
		})
		require.NoError(t, err)
		jobs := response.Jobs
		require.Equal(t, 1, len(jobs))
		require.Equal(t, "client1", jobs[0].Namespace)
		require.Contains(t, jobs[0].Labels, "fast")
		require.NotContains(t, jobs[0].Labels, "slow")
	})

	s.T().Run("basic selectors", func(t *testing.T) {
		response, err := s.store.GetJobs(s.ctx, jobstore.JobQuery{
			Namespace: "client1",
			Selector:  s.parseLabels("gpu=true,fast=true"),
		})
		require.NoError(t, err)
		jobs := response.Jobs
		require.Equal(t, 1, len(jobs))
		require.Equal(t, "client1", jobs[0].Namespace)
	})

	s.T().Run("all records with selectors and paging", func(t *testing.T) {
		response, err := s.store.GetJobs(s.ctx, jobstore.JobQuery{
			SortBy:   "created_at",
			Selector: s.parseLabels("max>1"),
			Limit:    2,
		})
		require.NoError(t, err)
		jobs := response.Jobs
		require.Equal(t, 2, len(jobs))

		// Having skipped the first two s.ids because of non-matching selectors,
		// we expect the next two to match
		require.Equal(t, "130", jobs[0].ID)
		require.Equal(t, "140", jobs[1].ID)

		response, err = s.store.GetJobs(s.ctx, jobstore.JobQuery{
			SortBy:   "created_at",
			Selector: s.parseLabels("max>1"),
			Limit:    2,
			Offset:   2,
		})

		require.NoError(t, err)
		require.Equal(t, 1, len(response.Jobs))
	})

	s.T().Run("everything sorted by created_at", func(t *testing.T) {
		response, err := s.store.GetJobs(s.ctx, jobstore.JobQuery{
			ReturnAll: true,
		})
		require.NoError(t, err)
		jobs := response.Jobs
		require.Equal(t, 5, len(jobs))
		ids := lo.Map(jobs, func(item models.Job, _ int) string {
			return item.ID
		})
		require.EqualValues(t, []string{"110", "120", "130", "140", "150"}, ids)

		response, err = s.store.GetJobs(s.ctx, jobstore.JobQuery{
			ReturnAll:   true,
			SortReverse: true,
		})
		require.NoError(t, err)
		jobs = response.Jobs
		require.Equal(t, 5, len(jobs))
		ids = lo.Map(jobs, func(item models.Job, _ int) string {
			return item.ID
		})
		require.EqualValues(t, []string{"150", "140", "130", "120", "110"}, ids)
	})

	s.T().Run("everything", func(t *testing.T) {
		response, err := s.store.GetJobs(s.ctx, jobstore.JobQuery{
			ReturnAll: true,
		})
		require.NoError(t, err)
		require.Equal(t, 5, len(response.Jobs))
	})

	s.T().Run("everything offset", func(t *testing.T) {
		response, err := s.store.GetJobs(s.ctx, jobstore.JobQuery{
			ReturnAll: true,
			Offset:    1,
		})
		require.NoError(t, err)
		require.Equal(t, 4, len(response.Jobs))
		require.Equal(t, uint32(1), response.Offset)
	})

	s.T().Run("everything limit", func(t *testing.T) {
		response, err := s.store.GetJobs(s.ctx, jobstore.JobQuery{
			ReturnAll: true,
			Limit:     2,
		})
		require.NoError(t, err)
		require.Equal(t, 2, len(response.Jobs))
	})

	s.T().Run("everything offset/limit", func(t *testing.T) {
		response, err := s.store.GetJobs(s.ctx, jobstore.JobQuery{
			ReturnAll: true,
			Offset:    1,
			Limit:     1,
		})
		require.NoError(t, err)
		require.Equal(t, 1, len(response.Jobs))
	})

	s.T().Run("include tags", func(t *testing.T) {
		response, err := s.store.GetJobs(s.ctx, jobstore.JobQuery{
			IncludeTags: []string{"gpu"},
		})
		require.NoError(t, err)
		require.Equal(t, 1, len(response.Jobs))
		require.Equal(t, "110", response.Jobs[0].ID)
	})

	s.T().Run("all but exclude tags", func(t *testing.T) {
		response, err := s.store.GetJobs(s.ctx, jobstore.JobQuery{
			ReturnAll:   true,
			ExcludeTags: []string{"fast"},
		})
		require.NoError(t, err)
		require.Equal(t, 4, len(response.Jobs))
	})

	s.T().Run("include/exclude same tag", func(t *testing.T) {
		response, err := s.store.GetJobs(s.ctx, jobstore.JobQuery{
			IncludeTags: []string{"gpu"},
			ExcludeTags: []string{"fast"},
		})
		require.NoError(t, err)
		require.Equal(t, 0, len(response.Jobs))
	})
}

func (s *BoltJobstoreTestSuite) TestDeleteJob() {
	job := makeDockerEngineJob(
		[]string{"bash", "-c", "echo hello"})
	job.Labels = map[string]string{"tag": "value"}
	job.ID = "deleteme"
	job.Namespace = "client1"

	err := s.store.CreateJob(s.ctx, *job)
	s.Require().NoError(err)

	err = s.store.DeleteJob(s.ctx, job.ID)
	s.Require().NoError(err)
}

func (s *BoltJobstoreTestSuite) TestGetJob() {
	job, err := s.store.GetJob(s.ctx, "110")
	s.Require().NoError(err)
	s.NotNil(job)

	_, err = s.store.GetJob(s.ctx, "100")
	s.Require().Error(err)
}

func (s *BoltJobstoreTestSuite) TestCreateExecution() {
	job := mock.Job()
	execution := mock.ExecutionForJob(job)
	s.Require().NoError(s.store.CreateJob(s.ctx, *job))
	s.Require().NoError(s.store.CreateExecution(s.ctx, *execution))

	// Ensure that the execution is created
	exec, err := s.store.GetExecutions(s.ctx, jobstore.GetExecutionsOptions{
		JobID: job.ID,
	})
	s.Require().NoError(err)
	s.Require().Equal(1, len(exec))
	s.Require().Nil(exec[0].Job)

	// Ensure that the execution is created and the job is included
	exec, err = s.store.GetExecutions(s.ctx, jobstore.GetExecutionsOptions{
		JobID:      job.ID,
		IncludeJob: true,
	})
	s.Require().NoError(err)
	s.Require().Equal(1, len(exec))
	s.Require().NotNil(exec[0].Job)
	s.Require().Equal(job.ID, exec[0].Job.ID)
}

func (s *BoltJobstoreTestSuite) TestGetExecutions() {
	state, err := s.store.GetExecutions(s.ctx, jobstore.GetExecutionsOptions{
		JobID: "110",
	})
	s.Require().NoError(err)
	s.NotNil(state)
	s.Equal(len(state), 1)
	s.Nil(state[0].Job)

	state, err = s.store.GetExecutions(s.ctx, jobstore.GetExecutionsOptions{
		JobID:      "110",
		IncludeJob: true,
	})
	s.Require().NoError(err)
	s.NotNil(state)
	s.Equal(len(state), 1)
	s.NotNil(state[0].Job)
	s.Equal("110", state[0].Job.ID)

	state, err = s.store.GetExecutions(s.ctx, jobstore.GetExecutionsOptions{
		JobID: "100",
	})
	s.Require().Error(err)
	s.Require().IsType(err, &bacerrors.JobNotFound{})
	s.Require().Nil(state)

	state, err = s.store.GetExecutions(s.ctx, jobstore.GetExecutionsOptions{
		JobID: "11",
	})
	s.Require().NoError(err)
	s.NotNil(state)
	s.Require().Equal("110", state[0].JobID)

	state, err = s.store.GetExecutions(s.ctx, jobstore.GetExecutionsOptions{
		JobID: "1",
	})
	s.Require().Error(err)
	s.Require().IsType(err, &bacerrors.MultipleJobsFound{})
	s.Require().Nil(state)

}

func (s *BoltJobstoreTestSuite) TestInProgressJobs() {
	infos, err := s.store.GetInProgressJobs(s.ctx, "")
	s.Require().NoError(err)
	s.Require().Equal(3, len(infos))
	s.Require().Equal("130", infos[0].ID)

	infos, err = s.store.GetInProgressJobs(s.ctx, "batch")
	s.Require().NoError(err)
	s.Require().Equal(2, len(infos))
	s.Require().Equal("130", infos[0].ID)

	infos, err = s.store.GetInProgressJobs(s.ctx, "daemon")
	s.Require().NoError(err)
	s.Require().Equal(1, len(infos))
	s.Require().Equal("150", infos[0].ID)
}

func (s *BoltJobstoreTestSuite) TestShortIDs() {
	uuidString := "9308d0d2-d93c-4e22-8a5b-c392e614922e"
	uuidString2 := "9308d0d2-d93c-4e22-8a5b-c392e614922f"
	shortString := "9308d0d2"

	job := makeDockerEngineJob(
		[]string{"bash", "-c", "echo hello"})
	job.ID = uuidString
	job.Namespace = "110"

	// No matches
	_, err := s.store.GetJob(s.ctx, shortString)
	s.Require().Error(err)
	s.Require().IsType(err, &bacerrors.JobNotFound{})

	// Create and fetch the single entry
	err = s.store.CreateJob(s.ctx, *job)
	s.Require().NoError(err)

	j, err := s.store.GetJob(s.ctx, shortString)
	s.Require().NoError(err)
	s.Require().Equal(uuidString, j.ID)

	// Add a record that will also match and expect an appropriate error
	job.ID = uuidString2
	err = s.store.CreateJob(s.ctx, *job)
	s.Require().NoError(err)

	_, err = s.store.GetJob(s.ctx, shortString)
	s.Require().Error(err)
	s.Require().IsType(err, &bacerrors.MultipleJobsFound{})
}

func (s *BoltJobstoreTestSuite) TestEvents() {
	watcher := s.store.Watch(s.ctx,
		jobstore.JobWatcher|jobstore.ExecutionWatcher,
		jobstore.CreateEvent|jobstore.UpdateEvent|jobstore.DeleteEvent,
	)

	job := makeDockerEngineJob(
		[]string{"bash", "-c", "echo hello"})
	job.ID = "10"
	job.Namespace = "110"

	var execution models.Execution

	s.Run("job create event", func() {
		err := s.store.CreateJob(s.ctx, *job)
		s.Require().NoError(err)

		// Read an event, it should be a jobcreate
		ev := <-watcher.Channel()
		s.Require().Equal(ev.Event, jobstore.CreateEvent)
		s.Require().Equal(ev.Kind, jobstore.JobWatcher)

		expectedJob, ok := ev.Object.(models.Job)
		s.Require().True(ok, "expected object to be a job")
		s.Require().Equal(expectedJob.ID, job.ID)
	})

	s.Run("execution create event", func() {
		s.clock.Add(1 * time.Second)
		execution = *mock.Execution()
		execution.JobID = "10"
		execution.ComputeState = models.State[models.ExecutionStateType]{StateType: models.ExecutionStateNew}
		err := s.store.CreateExecution(s.ctx, execution)
		s.Require().NoError(err)

		// Read an event, it should be a ExecutionForJob Create
		ev := <-watcher.Channel()
		s.Require().Equal(ev.Event, jobstore.CreateEvent)
		s.Require().Equal(ev.Kind, jobstore.ExecutionWatcher)
	})

	s.Run("update job state event", func() {
		request := jobstore.UpdateJobStateRequest{
			JobID:    "10",
			NewState: models.JobStateTypeRunning,
			Condition: jobstore.UpdateJobCondition{
				ExpectedState: models.JobStateTypePending,
			},
		}
		_ = s.store.UpdateJobState(s.ctx, request)
		ev := <-watcher.Channel()
		s.Require().Equal(ev.Event, jobstore.UpdateEvent)
		s.Require().Equal(ev.Kind, jobstore.JobWatcher)
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
		})
		ev := <-watcher.Channel()
		s.Require().Equal(ev.Event, jobstore.UpdateEvent)
		s.Require().Equal(ev.Kind, jobstore.ExecutionWatcher)

		expectedExec, ok := ev.Object.(models.Execution)
		s.Require().True(ok, "expected object to be an execution")
		s.Require().Equal(expectedExec.ID, execution.ID)
	})

	s.Run("delete job event", func() {
		_ = s.store.DeleteJob(s.ctx, job.ID)
		ev := <-watcher.Channel()
		s.Require().Equal(ev.Event, jobstore.DeleteEvent)
		s.Require().Equal(ev.Kind, jobstore.JobWatcher)
	})
}

func (s *BoltJobstoreTestSuite) TestEvaluations() {

	eval := models.Evaluation{
		ID:    "e1",
		JobID: "10",
	}

	// Wrong job ID means JobNotFound
	err := s.store.CreateEvaluation(s.ctx, eval)
	s.Require().Error(err)

	// Correct job ID
	eval.JobID = "110"
	err = s.store.CreateEvaluation(s.ctx, eval)
	s.Require().NoError(err)

	_, err = s.store.GetEvaluation(s.ctx, "missing")
	s.Require().Error(err)

	e, err := s.store.GetEvaluation(s.ctx, eval.ID)
	s.Require().NoError(err)
	s.Require().Equal(e, eval)

	err = s.store.DeleteEvaluation(s.ctx, eval.ID)
	s.Require().NoError(err)
}

// TestTransactionsWithTxContext tests the creation of transactional context
// and that multiple operations will be committed atomically with the context.
func (s *BoltJobstoreTestSuite) TestTransactionsWithTxContext() {
	txCtx, err := s.store.BeginTx(s.ctx)
	s.Require().NoError(err)
	s.Require().NotNil(txCtx)

	job := mock.Job()
	execution := mock.ExecutionForJob(job)
	evaluation := mock.EvalForJob(job)
	s.Require().NoError(s.store.CreateJob(txCtx, *job))
	s.Require().NoError(s.store.CreateExecution(txCtx, *execution))
	s.Require().NoError(s.store.CreateEvaluation(txCtx, *evaluation))

	// Commit the transaction
	s.Require().NoError(txCtx.Commit())

	// Ensure that the job is now available
	j, err := s.store.GetJob(s.ctx, job.ID)
	s.Require().NoError(err)
	s.Require().Equal(job.ID, j.ID)

	// Ensure that the execution is now available
	exec, err := s.store.GetExecutions(s.ctx, jobstore.GetExecutionsOptions{
		JobID:      job.ID,
		IncludeJob: true,
	})
	s.Require().NoError(err)
	s.Require().Equal(1, len(exec))
	s.Require().NotNil(exec[0].Job)
	s.Require().Equal(job.ID, exec[0].Job.ID)

	// Ensure that the evaluation is now available
	eval, err := s.store.GetEvaluation(s.ctx, evaluation.ID)
	s.Require().NoError(err)
	s.Require().Equal(evaluation.ID, eval.ID)
}

// TestTransactionsWithTxContextRollback tests the creation of transactional context
// and that multiple operations will be rolled back atomically with the context.
func (s *BoltJobstoreTestSuite) TestTransactionsWithTxContextRollback() {
	txCtx, err := s.store.BeginTx(s.ctx)
	s.Require().NoError(err)
	s.Require().NotNil(txCtx)

	job := mock.Job()
	execution := mock.ExecutionForJob(job)
	evaluation := mock.EvalForJob(job)
	s.Require().NoError(s.store.CreateJob(txCtx, *job))
	s.Require().NoError(s.store.CreateExecution(txCtx, *execution))
	s.Require().NoError(s.store.CreateEvaluation(txCtx, *evaluation))

	// Rollback the transaction
	s.Require().NoError(txCtx.Rollback())

	// Ensure that no jobs are returned as the tx is not committed
	_, err = s.store.GetJob(s.ctx, job.ID)
	s.Require().Error(err)

	// Ensure that no executions are returned as the tx is not committed
	_, err = s.store.GetExecutions(s.ctx, jobstore.GetExecutionsOptions{
		JobID: job.ID,
	})
	s.Require().Error(err)

	// Ensure no evaluation is returned as the tx is not committed
	_, err = s.store.GetEvaluation(s.ctx, evaluation.ID)
	s.Require().Error(err)
}

// TestTransactionsWithTxContextCancellation tests the creation of transactional context
// and that multiple operations will be rolled back atomically with the context cancellation
func (s *BoltJobstoreTestSuite) TestTransactionsWithTxContextCancellation() {
	ctx, cancel := context.WithCancel(s.ctx)
	txCtx, err := s.store.BeginTx(ctx)
	s.Require().NoError(err)
	s.Require().NotNil(txCtx)

	job := mock.Job()
	execution := mock.ExecutionForJob(job)
	evaluation := mock.EvalForJob(job)
	s.Require().NoError(s.store.CreateJob(txCtx, *job))
	s.Require().NoError(s.store.CreateExecution(txCtx, *execution))
	s.Require().NoError(s.store.CreateEvaluation(txCtx, *evaluation))

	// cancel the context
	cancel()
	<-txCtx.Done()

	// Ensure that no jobs are returned as the tx is not committed
	_, err = s.store.GetJob(s.ctx, job.ID)
	s.Require().Error(err)

	// Ensure that no executions are returned as the tx is not committed
	_, err = s.store.GetExecutions(s.ctx, jobstore.GetExecutionsOptions{
		JobID: job.ID,
	})
	s.Require().Error(err)

	// Ensure no evaluation is returned as the tx is not committed
	_, err = s.store.GetEvaluation(s.ctx, evaluation.ID)
	s.Require().Error(err)
}

// TestTransactionsReadDuringWrite tests we can read data that was written in the same transaction
func (s *BoltJobstoreTestSuite) TestTransactionsReadDuringWrite() {
	// Create a job outside the transaction
	oldJob := mock.Job()
	s.Require().NoError(s.store.CreateJob(s.ctx, *oldJob))

	txCtx, err := s.store.BeginTx(s.ctx)
	s.Require().NoError(err)
	s.Require().NotNil(txCtx)

	job := mock.Job()
	s.Require().NoError(s.store.CreateJob(txCtx, *job))

	// make sure we can read existing data during transaction
	readOldJob, err := s.store.GetJob(txCtx, oldJob.ID)
	s.Require().NoError(err)
	s.Require().Equal(oldJob.ID, readOldJob.ID)

	// make sure we can read uncommitted data during transaction
	readJob, err := s.store.GetJob(txCtx, job.ID)
	s.Require().NoError(err)
	s.Require().Equal(job.ID, readJob.ID)

	// Commit the transaction
	s.Require().NoError(txCtx.Commit())
}

func (s *BoltJobstoreTestSuite) TestBeginMultipleTransactions_Sequential() {
	txCtx1, err := s.store.BeginTx(s.ctx)
	s.Require().NoError(err)
	s.Require().NotNil(txCtx1)
	tx1, ok := txFromContext(txCtx1)
	s.Require().True(ok)
	// commit to release the transaction
	s.Require().NoError(txCtx1.Commit())

	// start second transaction, even through tcCtx1
	txCtx2, err := s.store.BeginTx(txCtx1)
	s.Require().NoError(err)
	s.Require().NotNil(txCtx2)
	tx2, ok := txFromContext(txCtx2)
	s.Require().True(ok)
	// commit to release the transaction
	s.Require().NoError(txCtx2.Commit())

	// assert that the two transactions were different
	s.Require().NotEqual(txCtx1, txCtx2)
	s.Require().NotEqual(tx1, tx2)
}

func (s *BoltJobstoreTestSuite) TestBeginMultipleTransactions_Concurrent() {
	// Start the first transaction
	txCtx1, err := s.store.BeginTx(s.ctx)
	s.Require().NoError(err)
	s.Require().NotNil(txCtx1)

	// Channel to signal when the second transaction attempt is complete
	done := make(chan bool)

	// Start a goroutine to attempt the second transaction
	var txCtx2 jobstore.TxContext
	go func() {
		txCtx2, err = s.store.BeginTx(s.ctx)
		s.Require().NoError(err)
		done <- true
	}()

	// Ensure the second transaction attempt has completed
	select {
	case <-done:
		s.Fail("The second transaction attempt should not have completed")
	case <-time.After(100 * time.Millisecond):
		// Success
	}

	// Commit the first transaction
	s.Require().NoError(txCtx1.Commit())
	select {
	case <-done:
		// Success, now commit the second transaction
		s.Require().NoError(txCtx2.Commit())
	case <-time.After(100 * time.Millisecond):
		s.Fail("The second transaction should've started")
	}
}

func (s *BoltJobstoreTestSuite) parseLabels(selector string) labels.Selector {
	req, err := labels.ParseToRequirements(selector)
	s.NoError(err)

	return labels.NewSelector().Add(req...)
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
