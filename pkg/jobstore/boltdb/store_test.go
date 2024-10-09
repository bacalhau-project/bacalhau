//go:build unit || !integration

package boltjobstore

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
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
		id         string
		jobType    string
		client     string
		tags       map[string]string
		jobStates  []models.JobStateType
		executions map[int][]models.ExecutionStateType
	}{
		{
			id:        "110",
			client:    "client1",
			jobType:   "batch",
			tags:      map[string]string{"gpu": "true", "fast": "true"},
			jobStates: []models.JobStateType{models.JobStateTypePending, models.JobStateTypeRunning, models.JobStateTypeStopped},
			executions: map[int][]models.ExecutionStateType{
				1: {models.ExecutionStateAskForBid, models.ExecutionStateAskForBidAccepted, models.ExecutionStateCancelled},
			},
		},
		{
			id:        "120",
			client:    "client2",
			jobType:   "batch",
			tags:      map[string]string{},
			jobStates: []models.JobStateType{models.JobStateTypePending, models.JobStateTypeRunning, models.JobStateTypeStopped},
			executions: map[int][]models.ExecutionStateType{
				1: {models.ExecutionStateAskForBid, models.ExecutionStateAskForBidAccepted, models.ExecutionStateCancelled},
			},
		},
		{
			id:        "130",
			client:    "client3",
			jobType:   "batch",
			tags:      map[string]string{"slow": "true", "max": "10"},
			jobStates: []models.JobStateType{models.JobStateTypePending, models.JobStateTypeRunning},
			executions: map[int][]models.ExecutionStateType{
				1: {models.ExecutionStateAskForBid, models.ExecutionStateAskForBidAccepted},
			},
		},
		{
			id:        "140",
			client:    "client4",
			jobType:   "batch",
			tags:      map[string]string{"max": "10"},
			jobStates: []models.JobStateType{models.JobStateTypePending, models.JobStateTypeRunning},
			executions: map[int][]models.ExecutionStateType{
				1: {models.ExecutionStateAskForBid, models.ExecutionStateAskForBidAccepted},
			},
		},
		{
			id:        "150",
			client:    "client5",
			jobType:   "daemon",
			tags:      map[string]string{"max": "10"},
			jobStates: []models.JobStateType{models.JobStateTypePending, models.JobStateTypeRunning},
			executions: map[int][]models.ExecutionStateType{
				1: {models.ExecutionStateAskForBid, models.ExecutionStateAskForBidAccepted},
			},
		},
		{
			id:        "160",
			client:    "client6",
			jobType:   "batch",
			tags:      map[string]string{"max": "10"},
			jobStates: []models.JobStateType{models.JobStateTypePending, models.JobStateTypeRunning},
			executions: map[int][]models.ExecutionStateType{
				1: {models.ExecutionStateAskForBid, models.ExecutionStateAskForBidAccepted, models.ExecutionStateFailed},
				2: {models.ExecutionStateAskForBid, models.ExecutionStateAskForBidAccepted, models.ExecutionStateCompleted},
			},
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

		for _, executionStates := range fixture.executions {
			s.clock.Add(1 * time.Second)
			execution := mock.ExecutionForJob(job)
			execution.ComputeState.StateType = models.ExecutionStateNew
			// clear out CreateTime and ModifyTime from the mocked execution to let the job store fill those
			execution.CreateTime = 0
			execution.ModifyTime = 0
			s.Require().NoError(s.store.CreateExecution(s.ctx, *execution))
			s.Require().NoError(s.store.AddExecutionHistory(s.ctx, fixture.id, execution.ID, *models.NewEvent("test").WithMessage("execution created")))

			for i, state := range executionStates {

				s.clock.Add(1 * time.Second)

				oldState := models.ExecutionStateNew
				if i > 0 {
					oldState = executionStates[i-1]
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
}

func (s *BoltJobstoreTestSuite) TearDownTest() {
	s.store.Close(s.ctx)
	os.Remove(s.dbFile)
}

func (s *BoltJobstoreTestSuite) TestUnfilteredJobHistory() {
	jobHistoryQueryResponse, err := s.store.GetJobHistory(s.ctx, "110", jobstore.JobHistoryQuery{})
	s.Require().NoError(err, "failed to get job history")
	s.Require().Equal(8, len(jobHistoryQueryResponse.JobHistory))

	jobHistoryQueryResponse, err = s.store.GetJobHistory(s.ctx, "11", jobstore.JobHistoryQuery{})
	s.Require().NoError(err)
	s.NotEmpty(jobHistoryQueryResponse)
	s.Require().Equal("110", jobHistoryQueryResponse.JobHistory[0].JobID)

	jobHistoryQueryResponse, err = s.store.GetJobHistory(s.ctx, "1", jobstore.JobHistoryQuery{})
	s.Require().Error(err)
	s.Require().True(bacerrors.IsError(err))
	s.Require().Nil(jobHistoryQueryResponse)
}

func (s *BoltJobstoreTestSuite) TestJobHistoryOrdering() {
	jobHistoryQueryResponse, err := s.store.GetJobHistory(s.ctx, "110", jobstore.JobHistoryQuery{})
	require.NoError(s.T(), err, "failed to get job history")

	// There are 6 history entries that we created directly, and 2 created by
	// CreateJob and CreateExecution
	require.Equal(s.T(), 8, len(jobHistoryQueryResponse.JobHistory))

	// Make sure they come back in order
	values := make([]int64, len(jobHistoryQueryResponse.JobHistory))
	for i, h := range jobHistoryQueryResponse.JobHistory {
		values[i] = h.Time.Unix()
		s.Require().Equal(uint64(i+1), h.SeqNum, "Sequence numbers should be in order")
	}

	require.Equal(s.T(), []int64{1, 2, 3, 4, 5, 6, 7, 8}, values)
}

func (s *BoltJobstoreTestSuite) TestJobHistoryOffset() {
	terminalJobID := "110"
	ongoingJobID := "130"

	testCases := []struct {
		name           string
		jobID          string
		offset         uint64
		expectedSeqNum uint64
		expectedNext   uint64
	}{
		{"Start from 0", ongoingJobID, 0, 1, 2},
		{"Start from 1", ongoingJobID, 1, 1, 2},
		{"Offset by 2", ongoingJobID, 2, 2, 3},
		{"Offset by 4", ongoingJobID, 4, 4, 5},
		{"Beyond the end", ongoingJobID, 10, 0, 10},

		{"Terminal job", terminalJobID, 0, 1, 2},
		{"Terminal job - offset by 2", terminalJobID, 2, 2, 3},
		{"Terminal job - beyond the end", terminalJobID, 10, 0, 0},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			query := jobstore.JobHistoryQuery{
				Limit: 1,
				NextToken: models.NewPagingToken(&models.PagingTokenParams{
					Offset: tc.offset,
					Limit:  1,
				}).String(),
			}

			response, err := s.store.GetJobHistory(s.ctx, tc.jobID, query)
			require.NoError(s.T(), err, "Failed to get job history")

			if tc.expectedSeqNum == 0 {
				require.Empty(s.T(), response.JobHistory, "No history item should be returned")
			} else {
				require.Len(s.T(), response.JobHistory, 1, "Should return exactly one history item")
				require.Equal(s.T(), tc.expectedSeqNum, response.JobHistory[0].SeqNum, "Sequence number should match expected")
			}

			// Check if NextToken is set correctly
			if tc.expectedNext == 0 {
				require.Empty(s.T(), response.NextToken, "NextToken should be empty")
			} else {
				require.NotEmpty(s.T(), response.NextToken, "NextToken should be set")
				token, err := models.NewPagingTokenFromString(response.NextToken)
				require.NoError(s.T(), err, "Failed to parse NextToken")
				require.Equal(s.T(), tc.expectedNext, token.Offset, "Next offset should be current offset + 1")
				require.Equal(s.T(), uint32(1), token.Limit, "Limit should be 1")
			}
		})
	}
}

func (s *BoltJobstoreTestSuite) TestJobHistoryPagination() {
	// Setup: Create two jobs - one ongoing and one terminal
	ongoingJob, terminalExec, ongoingExec := s.createJobWithHistory(false)
	ongoingJobEndTime := s.clock.Now()
	terminalJob, _, _ := s.createJobWithHistory(true)

	testCases := []struct {
		name       string
		jobID      string
		query      jobstore.JobHistoryQuery
		isTerminal bool
		pageSize   int
		expected   int
	}{
		{
			name:     "Ongoing job - all events",
			jobID:    ongoingJob,
			query:    jobstore.JobHistoryQuery{},
			pageSize: 5,
			expected: 15,
		},
		{
			name:     "Ongoing job - job-level events",
			jobID:    ongoingJob,
			query:    jobstore.JobHistoryQuery{ExcludeExecutionLevel: true},
			pageSize: 2,
			expected: 5,
		},
		{
			name:     "Ongoing job - execution-level events",
			jobID:    ongoingJob,
			query:    jobstore.JobHistoryQuery{ExcludeJobLevel: true},
			pageSize: 3,
			expected: 10,
		},
		{
			name:       "Terminal job - all events",
			jobID:      terminalJob,
			query:      jobstore.JobHistoryQuery{},
			pageSize:   4,
			expected:   15,
			isTerminal: true,
		},
		{
			name:       "Filter by terminal ExecutionID",
			jobID:      ongoingJob,
			query:      jobstore.JobHistoryQuery{ExecutionID: terminalExec},
			pageSize:   3,
			expected:   5,
			isTerminal: true,
		},
		{
			name:     "Filter by ongoing ExecutionID",
			jobID:    ongoingJob,
			query:    jobstore.JobHistoryQuery{ExecutionID: ongoingExec},
			pageSize: 5,
			expected: 5,
		},
		{
			name:     "Since timestamp",
			jobID:    ongoingJob,
			query:    jobstore.JobHistoryQuery{Since: ongoingJobEndTime.Add(-5 * time.Second).Unix()},
			pageSize: 10,
			expected: 6, // inclusive
		},
		{
			name:     "Combination of filters",
			jobID:    ongoingJob,
			query:    jobstore.JobHistoryQuery{ExecutionID: ongoingExec, ExcludeJobLevel: true, Since: ongoingJobEndTime.Add(-6 * time.Second).Unix()},
			pageSize: 2,
			expected: 3,
		},
		{
			name:     "Large page size",
			jobID:    ongoingJob,
			query:    jobstore.JobHistoryQuery{},
			pageSize: 100,
			expected: 15,
		},
		{
			name:     "Small page size",
			jobID:    ongoingJob,
			query:    jobstore.JobHistoryQuery{},
			pageSize: 1,
			expected: 15,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			var allEvents []models.JobHistory
			nextToken := ""
			queryCount := 0

			for {
				queryCount++
				query := tc.query
				query.Limit = uint32(tc.pageSize)
				query.NextToken = nextToken

				response, err := s.store.GetJobHistory(s.ctx, tc.jobID, query)
				s.Require().NoError(err, "Failed to get job history")
				s.Require().LessOrEqual(len(response.JobHistory), tc.pageSize, "Unexpected number of events")
				allEvents = append(allEvents, response.JobHistory...)

				if len(response.JobHistory) > 0 {
					s.Require().NotEqual(nextToken, response.NextToken, "NextToken should change if there are more events")
				}

				nextToken = response.NextToken
				if len(response.JobHistory) == 0 || len(allEvents) >= tc.expected {
					break
				}
			}

			// verify individual events
			for _, event := range allEvents {
				if tc.query.ExecutionID != "" {
					s.Require().Equal(tc.query.ExecutionID, event.ExecutionID, "Event does not match the requested ExecutionID")
				}
				if tc.query.Since != 0 {
					s.Require().GreaterOrEqual(event.Time.Unix(), tc.query.Since, "Event time %d does not match the requested Since %s", event.Time.Unix(), tc.query.Since)
				}
				if tc.query.ExcludeJobLevel {
					s.Require().Equal(models.JobHistoryTypeExecutionLevel, event.Type, "Unexpected event type for execution-level events")
				}
				if tc.query.ExcludeExecutionLevel {
					s.Require().Equal(models.JobHistoryTypeJobLevel, event.Type, "Unexpected event type for job-level events")
				}
			}

			// Verify the total number of events
			s.Require().Len(allEvents, tc.expected,
				"Unexpected total number of events. Expected %d, but got %d", tc.expected, len(allEvents))

			// Verify the number of queries
			expectedQueries := (tc.expected + tc.pageSize - 1) / tc.pageSize
			s.Require().Equal(expectedQueries, queryCount, "Unexpected number of queries")

			// if we have a next token, do one more query to ensure it's empty and nextToken doesn't move
			// this is the case when the job is not terminal and we've read all available events
			if nextToken != "" {
				query := tc.query
				query.Limit = uint32(tc.pageSize)
				query.NextToken = nextToken

				response, err := s.store.GetJobHistory(s.ctx, tc.jobID, query)
				s.Require().NoError(err, "Failed to get job history")
				s.Require().Empty(response.JobHistory, "Expected no more events")
				s.Require().Equal(nextToken, response.NextToken, "NextToken should not change if there are no more events")
			}

			if tc.isTerminal {
				s.Require().Empty(nextToken, "Terminal job should end with an empty NextToken")
			} else {
				s.Require().NotEmpty(nextToken, "Non-terminal job should end with a non-empty NextToken")
			}

			// Verify the order of events
			s.Require().True(sort.SliceIsSorted(allEvents, func(i, j int) bool {
				return allEvents[i].Time.Before(allEvents[j].Time)
			}), "Events are not in the correct order")

		})
	}
}

// Helper function to create a job with a specified number of events
func (s *BoltJobstoreTestSuite) createJobWithHistory(makeJobTerminal bool) (string, string, string) {
	job := mock.Job()
	s.Require().NoError(s.store.CreateJob(s.ctx, *job))

	// create two executions
	var executions []string
	for i := 0; i < 2; i++ {
		execution := mock.ExecutionForJob(job)
		execution.ID = fmt.Sprintf("%s-%d", job.ID, i)
		s.Require().NoError(s.store.CreateExecution(s.ctx, *execution))
		executions = append(executions, execution.ID)
	}

	// Add events
	eventCount := 15
	for i := 0; i < eventCount/3; i++ {
		s.clock.Add(time.Second)
		s.Require().NoError(s.store.AddJobHistory(s.ctx, job.ID, *models.NewEvent("job-event").WithMessage(fmt.Sprintf("Job event %d", i))))
		s.clock.Add(time.Second)
		s.Require().NoError(s.store.AddExecutionHistory(s.ctx, job.ID, executions[0], *models.NewEvent("exec-event").WithMessage(fmt.Sprintf("Execution event %d", i))))
		s.clock.Add(time.Second)
		s.Require().NoError(s.store.AddExecutionHistory(s.ctx, job.ID, executions[1], *models.NewEvent("exec-event").WithMessage(fmt.Sprintf("Execution event %d", i))))
	}

	// Make the first execution terminal
	s.Require().NoError(s.store.UpdateExecution(s.ctx, jobstore.UpdateExecutionRequest{
		ExecutionID: executions[0],
		NewValues: models.Execution{
			JobID:        job.ID,
			ComputeState: models.NewExecutionState(models.ExecutionStateCompleted),
		},
	}))

	if makeJobTerminal {
		s.Require().NoError(s.store.UpdateJobState(s.ctx, jobstore.UpdateJobStateRequest{
			JobID:    job.ID,
			NewState: models.JobStateTypeCompleted,
		}))
	}

	return job.ID, executions[0], executions[1]
}

func (s *BoltJobstoreTestSuite) TestTimeFilteredJobHistory() {
	options := jobstore.JobHistoryQuery{
		Since: 5,
	}

	jobHistoryQueryResponse, err := s.store.GetJobHistory(s.ctx, "110", options)
	require.NoError(s.T(), err, "failed to get job history")
	require.Equal(s.T(), 4, len(jobHistoryQueryResponse.JobHistory))
}

func (s *BoltJobstoreTestSuite) TestExecutionFilteredJobHistory() {
	jobHistoryQueryResponse, err := s.store.GetJobHistory(s.ctx, "110", jobstore.JobHistoryQuery{})
	require.NoError(s.T(), err)

	var executionID string
	for _, h := range jobHistoryQueryResponse.JobHistory {
		if h.ExecutionID != "" {
			executionID = h.ExecutionID
			break
		}
	}
	require.NotEmpty(s.T(), executionID, "failed to find execution ID")

	options := jobstore.JobHistoryQuery{
		ExecutionID: executionID,
	}

	jobHistoryQueryResponse, err = s.store.GetJobHistory(s.ctx, "110", options)
	require.NoError(s.T(), err, "failed to get job history")

	for _, h := range jobHistoryQueryResponse.JobHistory {
		require.Equal(s.T(), executionID, h.ExecutionID)
	}
}

func (s *BoltJobstoreTestSuite) TestLevelFilteredJobHistory() {
	jobOptions := jobstore.JobHistoryQuery{
		ExcludeExecutionLevel: true,
	}
	execOptions := jobstore.JobHistoryQuery{
		ExcludeJobLevel: true,
	}

	jobHistoryQueryResponse, err := s.store.GetJobHistory(s.ctx, "110", jobOptions)
	s.Require().NoError(err, "failed to get job history")
	s.Require().Equal(4, len(jobHistoryQueryResponse.JobHistory))

	count := lo.Reduce(jobHistoryQueryResponse.JobHistory, func(agg int, item models.JobHistory, _ int) int {
		if item.Type == models.JobHistoryTypeJobLevel {
			return agg + 1
		}
		return agg
	}, 0)
	s.Require().Equal(count, 4)

	jobHistoryQueryResponse, err = s.store.GetJobHistory(s.ctx, "110", execOptions)
	s.Require().NoError(err, "failed to get job history")
	s.Require().Equal(4, len(jobHistoryQueryResponse.JobHistory))

	count = lo.Reduce(jobHistoryQueryResponse.JobHistory, func(agg int, item models.JobHistory, _ int) int {
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
		require.Equal(t, 2, len(response.Jobs))
	})

	s.T().Run("everything sorted by created_at", func(t *testing.T) {
		response, err := s.store.GetJobs(s.ctx, jobstore.JobQuery{
			ReturnAll: true,
		})
		require.NoError(t, err)
		jobs := response.Jobs
		require.Equal(t, 6, len(jobs))
		ids := lo.Map(jobs, func(item models.Job, _ int) string {
			return item.ID
		})
		require.EqualValues(t, []string{"110", "120", "130", "140", "150", "160"}, ids)

		response, err = s.store.GetJobs(s.ctx, jobstore.JobQuery{
			ReturnAll:   true,
			SortReverse: true,
		})
		require.NoError(t, err)
		jobs = response.Jobs
		require.Equal(t, 6, len(jobs))
		ids = lo.Map(jobs, func(item models.Job, _ int) string {
			return item.ID
		})
		require.EqualValues(t, []string{"160", "150", "140", "130", "120", "110"}, ids)
	})

	s.T().Run("everything", func(t *testing.T) {
		response, err := s.store.GetJobs(s.ctx, jobstore.JobQuery{
			ReturnAll: true,
		})
		require.NoError(t, err)
		require.Equal(t, 6, len(response.Jobs))
	})

	s.T().Run("everything offset", func(t *testing.T) {
		response, err := s.store.GetJobs(s.ctx, jobstore.JobQuery{
			ReturnAll: true,
			Offset:    1,
		})
		require.NoError(t, err)
		require.Equal(t, 5, len(response.Jobs))
		require.Equal(t, uint64(1), response.Offset)
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
		require.Equal(t, 5, len(response.Jobs))
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
	s.Equal(1, len(state))
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
	s.Require().True(bacerrors.IsError(err))
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
	s.Require().True(bacerrors.IsError(err))
	s.Require().Nil(state)

	// Created At Ascending Order Sort
	state, err = s.store.GetExecutions(s.ctx, jobstore.GetExecutionsOptions{
		JobID:   "160",
		OrderBy: "created_at",
	})
	s.Require().NoError(err)
	s.NotNil(state)
	s.Equal(2, len(state))
	s.Equal(state[0].GetCreateTime().Before(state[1].GetCreateTime()), true)

	// Created At Descending Order Sort
	state, err = s.store.GetExecutions(s.ctx, jobstore.GetExecutionsOptions{
		JobID:   "160",
		OrderBy: "created_at",
		Reverse: true,
	})
	s.Require().NoError(err)
	s.NotNil(state)
	s.Equal(2, len(state))
	s.Equal(state[0].GetCreateTime().After(state[1].GetCreateTime()), true)

	// Created Time Backward Compatibility Ascending Order Sort
	state, err = s.store.GetExecutions(s.ctx, jobstore.GetExecutionsOptions{
		JobID:   "160",
		OrderBy: "create_time",
	})
	s.Require().NoError(err)
	s.NotNil(state)
	s.Equal(2, len(state))
	s.Equal(state[0].GetCreateTime().Before(state[1].GetCreateTime()), true)

	// Create Time Backward Compatibility Descending Order Sort
	state, err = s.store.GetExecutions(s.ctx, jobstore.GetExecutionsOptions{
		JobID:   "160",
		OrderBy: "create_time",
		Reverse: true,
	})
	s.Require().NoError(err)
	s.NotNil(state)
	s.Equal(2, len(state))
	s.Equal(state[0].GetCreateTime().After(state[1].GetCreateTime()), true)

	// When OrderBy Empty, Created At Used as Default
	state, err = s.store.GetExecutions(s.ctx, jobstore.GetExecutionsOptions{
		JobID: "160",
	})
	s.Require().NoError(err)
	s.NotNil(state)
	s.Equal(2, len(state))
	s.Equal(state[0].GetCreateTime().Before(state[1].GetCreateTime()), true)

	// When OrderBy is set to Modified At
	state, err = s.store.GetExecutions(s.ctx, jobstore.GetExecutionsOptions{
		JobID:   "160",
		OrderBy: "modified_at",
	})
	s.Require().NoError(err)
	s.NotNil(state)
	s.Equal(2, len(state))
	s.Equal(state[0].GetModifyTime().Before(state[1].GetModifyTime()), true)

	// When OrderBy is set to Modified At With Reverse
	state, err = s.store.GetExecutions(s.ctx, jobstore.GetExecutionsOptions{
		JobID:   "160",
		OrderBy: "modified_at",
		Reverse: true,
	})
	s.Require().NoError(err)
	s.NotNil(state)
	s.Equal(2, len(state))
	s.Equal(state[0].GetModifyTime().After(state[1].GetModifyTime()), true)

	// When OrderBy is set to Modify Time (Backward Compatibility)
	state, err = s.store.GetExecutions(s.ctx, jobstore.GetExecutionsOptions{
		JobID:   "160",
		OrderBy: "modify_time",
	})
	s.Require().NoError(err)
	s.NotNil(state)
	s.Equal(2, len(state))
	s.Equal(state[0].GetModifyTime().Before(state[1].GetModifyTime()), true)

	// When OrderBy is set to Modify Time (Backward Compatibility)
	state, err = s.store.GetExecutions(s.ctx, jobstore.GetExecutionsOptions{
		JobID:   "160",
		OrderBy: "modify_time",
		Reverse: true,
	})
	s.Require().NoError(err)
	s.NotNil(state)
	s.Equal(2, len(state))
	s.Equal(state[0].GetModifyTime().After(state[1].GetModifyTime()), true)
}

func (s *BoltJobstoreTestSuite) TestInProgressJobs() {
	infos, err := s.store.GetInProgressJobs(s.ctx, "")
	s.Require().NoError(err)
	s.Require().Equal(4, len(infos))
	s.Require().Equal("130", infos[0].ID)

	infos, err = s.store.GetInProgressJobs(s.ctx, "batch")
	s.Require().NoError(err)
	s.Require().Equal(3, len(infos))
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
	s.Require().True(bacerrors.IsError(err))

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
	s.Require().True(bacerrors.IsError(err))
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
