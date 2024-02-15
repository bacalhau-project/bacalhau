package sqlite_test

import (
	"context"
	"encoding/json"
	"fmt"
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
	"github.com/bacalhau-project/bacalhau/pkg/jobstore/sqlite"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/test/mock"
)

type JobstoreTestSuite struct {
	suite.Suite
	store  *sqlite.Store
	dbFile string
	ctx    context.Context
	clock  *clock.Mock
}

func TestJobstoreTestSuite(t *testing.T) {
	suite.Run(t, new(JobstoreTestSuite))
}

func (s *JobstoreTestSuite) SetupTest() {
	s.clock = clock.NewMock()

	dir, _ := os.MkdirTemp("", "bacalhau-executionstore")
	s.dbFile = filepath.Join(dir, "test.boltdb")

	s.store, _ = sqlite.New()
	s.ctx = context.Background()

	jobFixtures := []struct {
		id              string
		client          string
		tags            map[string]string
		jobStates       []models.JobStateType
		executionStates []models.ExecutionStateType
	}{
		{
			id:              "110",
			client:          "client1",
			tags:            map[string]string{"gpu": "true", "fast": "true"},
			jobStates:       []models.JobStateType{models.JobStateTypePending, models.JobStateTypeRunning, models.JobStateTypeStopped},
			executionStates: []models.ExecutionStateType{models.ExecutionStateAskForBid, models.ExecutionStateAskForBidAccepted, models.ExecutionStateCancelled},
		},
		{
			id:              "120",
			client:          "client2",
			tags:            map[string]string{},
			jobStates:       []models.JobStateType{models.JobStateTypePending, models.JobStateTypeRunning, models.JobStateTypeStopped},
			executionStates: []models.ExecutionStateType{models.ExecutionStateAskForBid, models.ExecutionStateAskForBidAccepted, models.ExecutionStateCancelled},
		},
		{
			id:              "130",
			client:          "client3",
			tags:            map[string]string{"slow": "true", "max": "10"},
			jobStates:       []models.JobStateType{models.JobStateTypePending, models.JobStateTypeRunning},
			executionStates: []models.ExecutionStateType{models.ExecutionStateAskForBid, models.ExecutionStateAskForBidAccepted},
		},
		{
			id:              "140",
			client:          "client4",
			tags:            map[string]string{"max": "10"},
			jobStates:       []models.JobStateType{models.JobStateTypePending, models.JobStateTypeRunning},
			executionStates: []models.ExecutionStateType{models.ExecutionStateAskForBid, models.ExecutionStateAskForBidAccepted},
		},
		{
			id:              "150",
			client:          "client5",
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
		job.Labels = fixture.tags
		job.Namespace = fixture.client
		err := s.store.CreateJob(s.ctx, *job)
		s.Require().NoError(err)

		s.clock.Add(1 * time.Second)
		execution := mock.ExecutionForJob(job)
		execution.ComputeState.StateType = models.ExecutionStateNew
		err = s.store.CreateExecution(s.ctx, *execution)
		s.Require().NoError(err)

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
			s.Require().NoError(err)
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
			s.Require().NoError(err)
		}

	}
}

func (s *JobstoreTestSuite) TearDownTest() {
	err := s.store.DB.Exec(`
delete
from evaluations;

delete
from execution_states;

delete
from executions;

delete
from input_sources;

delete
from job_states;

delete
from network_configs;

delete
from resource_configs;

delete
from result_paths;

delete
from run_command_results;

delete
from spec_configs;

delete
from sqlite_sequence;

delete
from tasks;

delete
from jobs;

delete
from timeout_configs;
`).Error
	require.NoError(s.T(), err)
	s.store.Close(s.ctx)
	os.Remove(s.dbFile)
}

func (s *JobstoreTestSuite) TestUnfilteredJobHistory() {
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

func (s *JobstoreTestSuite) TestJobHistoryOrdering() {
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

func (s *JobstoreTestSuite) TestTimeFilteredJobHistory() {
	options := jobstore.JobHistoryFilterOptions{
		Since: 5,
	}

	history, err := s.store.GetJobHistory(s.ctx, "110", options)
	require.NoError(s.T(), err, "failed to get job history")
	require.Equal(s.T(), 4, len(history))
}

func (s *JobstoreTestSuite) TestExecutionFilteredJobHistory() {
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

func (s *JobstoreTestSuite) TestNodeFilteredJobHistory() {
	allHistories, err := s.store.GetJobHistory(s.ctx, "110", jobstore.JobHistoryFilterOptions{})
	require.NoError(s.T(), err)

	var nodeID string
	for _, h := range allHistories {
		if h.NodeID != "" {
			nodeID = h.NodeID
			break
		}
	}
	require.NotEmpty(s.T(), nodeID, "failed to find node ID")

	options := jobstore.JobHistoryFilterOptions{
		NodeID: nodeID,
	}

	history, err := s.store.GetJobHistory(s.ctx, "110", options)
	require.NoError(s.T(), err, "failed to get job history")

	for _, h := range history {
		require.Equal(s.T(), nodeID, h.NodeID)
	}
}

func (s *JobstoreTestSuite) TestLevelFilteredJobHistory() {
	jobOptions := jobstore.JobHistoryFilterOptions{
		ExcludeExecutionLevel: true,
	}
	execOptions := jobstore.JobHistoryFilterOptions{
		ExcludeJobLevel: true,
	}

	history, err := s.store.GetJobHistory(s.ctx, "110", jobOptions)
	s.Require().NoError(err, "failed to get job history")
	s.Require().Equal(4, len(history))
	s.Require().Equal(models.JobStateTypePending, history[1].JobState.New)

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
	s.Require().Equal(models.ExecutionStateAskForBid, history[1].ExecutionState.New)

	count = lo.Reduce(history, func(agg int, item models.JobHistory, _ int) int {
		if item.Type == models.JobHistoryTypeExecutionLevel {
			return agg + 1
		}
		return agg
	}, 0)
	s.Require().Equal(count, 4)
}

func (s *JobstoreTestSuite) TestSearchJobs() {
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

func (s *JobstoreTestSuite) TestDeleteJob() {
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

func (s *JobstoreTestSuite) TestGetJob() {
	job, err := s.store.GetJob(s.ctx, "110")
	s.Require().NoError(err)
	s.NotNil(job)

	_, err = s.store.GetJob(s.ctx, "100")
	s.Require().Error(err)
}

func (s *JobstoreTestSuite) TestCreateExecution() {
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

func (s *JobstoreTestSuite) TestGetExecutions() {
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

func (s *JobstoreTestSuite) TestInProgressJobs() {
	infos, err := s.store.GetInProgressJobs(s.ctx)
	s.Require().NoError(err)
	s.Require().Equal(3, len(infos))
	s.Require().Equal("130", infos[0].ID)
}

func (s *JobstoreTestSuite) TestShortIDs() {
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

func (s *JobstoreTestSuite) TestEvents() {
	ch := s.store.Watch(s.ctx,
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
		ev := <-ch
		s.Require().Equal(ev.Event, jobstore.CreateEvent)
		s.Require().Equal(ev.Kind, jobstore.JobWatcher)

		var decodedJob models.Job
		err = json.Unmarshal(ev.Object, &decodedJob)
		s.Require().NoError(err)
		s.Require().Equal(decodedJob.ID, job.ID)
	})

	s.Run("execution create event", func() {
		s.clock.Add(1 * time.Second)
		execution = *mock.Execution()
		execution.JobID = "10"
		execution.ComputeState = models.State[models.ExecutionStateType]{StateType: models.ExecutionStateNew}
		err := s.store.CreateExecution(s.ctx, execution)
		s.Require().NoError(err)

		// Read an event, it should be a ExecutionForJob Create
		ev := <-ch
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
			Comment: "event test",
		}
		_ = s.store.UpdateJobState(s.ctx, request)
		ev := <-ch
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
			Comment:   "event test",
		})
		ev := <-ch
		s.Require().Equal(ev.Event, jobstore.UpdateEvent)
		s.Require().Equal(ev.Kind, jobstore.ExecutionWatcher)

		var decodedExecution models.Execution
		err := json.Unmarshal(ev.Object, &decodedExecution)
		s.Require().NoError(err)
		s.Require().Equal(decodedExecution.ID, execution.ID)
	})

	s.Run("delete job event", func() {
		_ = s.store.DeleteJob(s.ctx, job.ID)
		ev := <-ch
		s.Require().Equal(ev.Event, jobstore.DeleteEvent)
		s.Require().Equal(ev.Kind, jobstore.JobWatcher)
	})
}

func (s *JobstoreTestSuite) TestEvaluations() {

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

func (s *JobstoreTestSuite) parseLabels(selector string) labels.Selector {
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
