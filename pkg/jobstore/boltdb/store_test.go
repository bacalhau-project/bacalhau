//go:build unit || !integration

package boltjobstore

import (
	"context"
	"errors"
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
	bolt "go.etcd.io/bbolt"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"

	"github.com/bacalhau-project/bacalhau/pkg/bacerrors"
	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/lib/boltdblib"
	"github.com/bacalhau-project/bacalhau/pkg/lib/watcher"
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

// ExecutionData represents the data for an execution
type ExecutionData struct {
	NodeID string
	States []models.ExecutionStateType
}

// JobVersionData represents the data for a specific version of a job
type JobVersionData struct {
	Version    uint64
	JobStates  []models.JobStateType
	Executions []ExecutionData
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
		id        string
		jobType   string
		namespace string
		tags      map[string]string
		versions  []JobVersionData
	}{
		{
			id:        "110",
			namespace: "client1",
			jobType:   "batch",
			tags:      map[string]string{"gpu": "true", "fast": "true"},
			versions: []JobVersionData{
				{
					Version:   1,
					JobStates: []models.JobStateType{models.JobStateTypeRunning, models.JobStateTypeStopped},
					Executions: []ExecutionData{
						{
							NodeID: "node-1",
							States: []models.ExecutionStateType{models.ExecutionStateAskForBid, models.ExecutionStateAskForBidAccepted, models.ExecutionStateCancelled},
						},
					},
				},
			},
		},
		{
			id:        "120",
			namespace: "client2",
			jobType:   "batch",
			tags:      map[string]string{},
			versions: []JobVersionData{
				{
					Version:   1,
					JobStates: []models.JobStateType{models.JobStateTypeRunning, models.JobStateTypeStopped},
					Executions: []ExecutionData{
						{
							NodeID: "node-1",
							States: []models.ExecutionStateType{models.ExecutionStateAskForBid, models.ExecutionStateAskForBidAccepted, models.ExecutionStateCancelled},
						},
					},
				},
			},
		},
		{
			id:        "130",
			namespace: "client3",
			jobType:   "batch",
			tags:      map[string]string{"slow": "true", "max": "10"},
			versions: []JobVersionData{
				{
					Version:   1,
					JobStates: []models.JobStateType{models.JobStateTypeRunning},
					Executions: []ExecutionData{
						{
							NodeID: "node-1",
							States: []models.ExecutionStateType{models.ExecutionStateAskForBid, models.ExecutionStateAskForBidAccepted},
						},
					},
				},
			},
		},
		{
			id:        "140",
			namespace: "client4",
			jobType:   "batch",
			tags:      map[string]string{"max": "10"},
			versions: []JobVersionData{
				{
					Version:   1,
					JobStates: []models.JobStateType{models.JobStateTypeRunning},
					Executions: []ExecutionData{
						{
							NodeID: "node-1",
							States: []models.ExecutionStateType{models.ExecutionStateAskForBid, models.ExecutionStateAskForBidAccepted},
						},
					},
				},
			},
		},
		{
			id:        "150",
			namespace: "client5",
			jobType:   "daemon",
			tags:      map[string]string{"max": "10"},
			versions: []JobVersionData{
				{
					Version:   1,
					JobStates: []models.JobStateType{models.JobStateTypeRunning},
					Executions: []ExecutionData{
						{
							NodeID: "node-1",
							States: []models.ExecutionStateType{models.ExecutionStateAskForBid, models.ExecutionStateAskForBidAccepted},
						},
					},
				},
			},
		},
		{
			id:        "160",
			namespace: "client6",
			jobType:   "batch",
			tags:      map[string]string{"max": "10"},
			versions: []JobVersionData{
				{
					Version:   1,
					JobStates: []models.JobStateType{models.JobStateTypeRunning},
					Executions: []ExecutionData{
						{
							NodeID: "node-1",
							States: []models.ExecutionStateType{models.ExecutionStateAskForBid, models.ExecutionStateAskForBidAccepted, models.ExecutionStateFailed},
						},
						{
							NodeID: "node-2",
							States: []models.ExecutionStateType{models.ExecutionStateAskForBid, models.ExecutionStateAskForBidAccepted, models.ExecutionStateCompleted},
						},
					},
				},
			},
		},
		{
			id:        "170",
			jobType:   "batch",
			namespace: "client7",
			versions: []JobVersionData{
				{
					Version:   1,
					JobStates: []models.JobStateType{models.JobStateTypeRunning, models.JobStateTypeStopped},
					Executions: []ExecutionData{
						{
							NodeID: "node-1",
							States: []models.ExecutionStateType{models.ExecutionStateAskForBid, models.ExecutionStateAskForBidAccepted},
						},
						{
							NodeID: "node-2",
							States: []models.ExecutionStateType{models.ExecutionStateAskForBid, models.ExecutionStateAskForBidAccepted, models.ExecutionStateCompleted},
						},
						{
							NodeID: "node-3",
							States: []models.ExecutionStateType{models.ExecutionStateAskForBid, models.ExecutionStateCancelled},
						},
					},
				},
				{
					Version:   2,
					JobStates: []models.JobStateType{models.JobStateTypeRunning},
					Executions: []ExecutionData{
						{
							NodeID: "node-1",
							States: []models.ExecutionStateType{models.ExecutionStateAskForBid},
						},
						{
							NodeID: "node-2",
							States: []models.ExecutionStateType{models.ExecutionStateAskForBid, models.ExecutionStateAskForBidAccepted},
						},
						{
							NodeID: "node-3",
							States: []models.ExecutionStateType{models.ExecutionStateAskForBid, models.ExecutionStateAskForBidAccepted, models.ExecutionStateCompleted},
						},
						{
							NodeID: "node-4",
							States: []models.ExecutionStateType{models.ExecutionStateAskForBid, models.ExecutionStateAskForBidAccepted, models.ExecutionStateFailed},
						},
						{
							NodeID: "node-5",
							States: []models.ExecutionStateType{models.ExecutionStateAskForBid, models.ExecutionStateCancelled},
						},
					},
				},
			},
		},
	}

	for _, fixture := range jobFixtures {
		s.clock.Add(1 * time.Second)
		job := makeDockerEngineJob(
			[]string{"sh", "-c", "echo hello"})

		job.ID = fixture.id
		job.Type = fixture.jobType
		job.Labels = fixture.tags
		job.Namespace = fixture.namespace
		s.Require().NoError(s.store.CreateJob(s.ctx, *job))
		s.Require().NoError(s.store.AddJobHistory(
			s.ctx,
			fixture.id,
			job.Version,
			*models.NewEvent("test").WithMessage("job created")),
		)

		revision := uint64(1)
		for i, versionData := range fixture.versions {
			// For version 1, we already created the job above, no need to update
			if i > 0 {
				// For versions beyond 1, use UpdateJob
				updatedJob := *job
				s.clock.Add(1 * time.Second)
				s.Require().NoError(s.store.UpdateJob(s.ctx, updatedJob))
				s.Require().NoError(s.store.AddJobHistory(
					s.ctx,
					fixture.id,
					versionData.Version,
					*models.NewEvent("test").WithMessage(fmt.Sprintf("job updated to version %d", versionData.Version))),
				)
			}

			for i, state := range versionData.JobStates {
				s.clock.Add(1 * time.Second)

				oldState := models.JobStateTypePending
				if i > 0 {
					oldState = versionData.JobStates[i-1]
				}

				request := jobstore.UpdateJobStateRequest{
					JobID:    fixture.id,
					NewState: state,
					Condition: jobstore.UpdateJobCondition{
						ExpectedState:    oldState,
						ExpectedRevision: revision,
					},
				}

				s.Require().NoErrorf(s.store.UpdateJobState(s.ctx, request),
					"Failed to update job state for job %s with version %d to %s", fixture.id, versionData.Version, state)
				s.Require().NoError(s.store.AddJobHistory(
					s.ctx,
					fixture.id,
					versionData.Version,
					*models.NewEvent("test").WithMessage(state.String())),
				)
				revision++
			}

			for _, executionData := range versionData.Executions {
				s.clock.Add(1 * time.Second)
				execution := mock.ExecutionForJob(job)
				execution.ComputeState.StateType = models.ExecutionStateNew
				// clear out CreateTime and ModifyTime from the mocked execution to let the job store fill those
				execution.CreateTime = 0
				execution.ModifyTime = 0
				execution.NodeID = executionData.NodeID
				execution.JobVersion = versionData.Version
				s.Require().NoError(s.store.CreateExecution(s.ctx, *execution))
				s.Require().NoError(s.store.AddExecutionHistory(s.ctx, fixture.id, versionData.Version, execution.ID, *models.NewEvent("test").WithMessage("execution created")))

				for i, state := range executionData.States {
					s.clock.Add(1 * time.Second)

					oldState := models.ExecutionStateNew
					if i > 0 {
						oldState = executionData.States[i-1]
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
					s.Require().NoError(s.store.AddExecutionHistory(s.ctx, fixture.id, versionData.Version, execution.ID, *models.NewEvent("test").WithMessage(state.String())))
				}
			}
		}
	}
}

func (s *BoltJobstoreTestSuite) TearDownTest() {
	s.store.Close(s.ctx)
	os.Remove(s.dbFile)
}

func (s *BoltJobstoreTestSuite) TestUnfilteredJobHistory() {
	jobHistoryQueryResponse, err := s.store.GetJobHistory(s.ctx, "110", jobstore.JobHistoryQuery{
		AllJobVersions: true,
	})
	s.Require().NoError(err, "failed to get job history")
	s.Require().Equal(7, len(jobHistoryQueryResponse.JobHistory))

	jobHistoryQueryResponse, err = s.store.GetJobHistory(s.ctx, "11", jobstore.JobHistoryQuery{
		AllJobVersions: true,
	})
	s.Require().NoError(err)
	s.NotEmpty(jobHistoryQueryResponse)
	s.Require().Equal("110", jobHistoryQueryResponse.JobHistory[0].JobID)

	jobHistoryQueryResponse, err = s.store.GetJobHistory(s.ctx, "1", jobstore.JobHistoryQuery{
		AllJobVersions: true,
	})
	s.Require().Error(err)
	s.Require().True(bacerrors.IsError(err))
	s.Require().Nil(jobHistoryQueryResponse)
}

func (s *BoltJobstoreTestSuite) TestJobHistoryOrdering() {
	jobHistoryQueryResponse, err := s.store.GetJobHistory(s.ctx, "110", jobstore.JobHistoryQuery{
		AllJobVersions: true,
	})
	require.NoError(s.T(), err, "failed to get job history")

	// There are 6 history entries that we created directly, and 2 created by
	// CreateJob and CreateExecution
	require.Equal(s.T(), 7, len(jobHistoryQueryResponse.JobHistory))

	// Make sure they come back in order
	values := make([]int64, len(jobHistoryQueryResponse.JobHistory))
	for i, h := range jobHistoryQueryResponse.JobHistory {
		values[i] = h.Time.Unix()
		s.Require().Equal(uint64(i+1), h.SeqNum, "Sequence numbers should be in order")
	}

	require.Equal(s.T(), []int64{1, 2, 3, 4, 5, 6, 7}, values)
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
				AllJobVersions: true,
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
				query.AllJobVersions = true

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
		s.Require().NoError(s.store.AddJobHistory(s.ctx, job.ID, job.Version, *models.NewEvent("job-event").WithMessage(fmt.Sprintf("Job event %d", i))))
		s.clock.Add(time.Second)
		s.Require().NoError(s.store.AddExecutionHistory(s.ctx, job.ID, job.Version, executions[0], *models.NewEvent("exec-event").WithMessage(fmt.Sprintf("Execution event %d", i))))
		s.clock.Add(time.Second)
		s.Require().NoError(s.store.AddExecutionHistory(s.ctx, job.ID, job.Version, executions[1], *models.NewEvent("exec-event").WithMessage(fmt.Sprintf("Execution event %d", i))))
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
		Since:          5,
		AllJobVersions: true,
	}

	jobHistoryQueryResponse, err := s.store.GetJobHistory(s.ctx, "110", options)
	require.NoError(s.T(), err, "failed to get job history")
	require.Equal(s.T(), 3, len(jobHistoryQueryResponse.JobHistory))
}

func (s *BoltJobstoreTestSuite) TestExecutionFilteredJobHistory() {
	jobHistoryQueryResponse, err := s.store.GetJobHistory(s.ctx, "110",
		jobstore.JobHistoryQuery{
			AllJobVersions: true,
		},
	)
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
		AllJobVersions:        true,
	}
	execOptions := jobstore.JobHistoryQuery{
		ExcludeJobLevel: true,
		AllJobVersions:  true,
	}

	jobHistoryQueryResponse, err := s.store.GetJobHistory(s.ctx, "110", jobOptions)
	s.Require().NoError(err, "failed to get job history")
	s.Require().Equal(3, len(jobHistoryQueryResponse.JobHistory))

	count := lo.Reduce(jobHistoryQueryResponse.JobHistory, func(agg int, item models.JobHistory, _ int) int {
		if item.Type == models.JobHistoryTypeJobLevel {
			return agg + 1
		}
		return agg
	}, 0)
	s.Require().Equal(count, 3)

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
		require.Equal(t, 7, len(jobs))
		ids := lo.Map(jobs, func(item models.Job, _ int) string {
			return item.ID
		})
		require.EqualValues(t, []string{"110", "120", "130", "140", "150", "160", "170"}, ids)

		response, err = s.store.GetJobs(s.ctx, jobstore.JobQuery{
			ReturnAll:   true,
			SortReverse: true,
		})
		require.NoError(t, err)
		jobs = response.Jobs
		require.Equal(t, 7, len(jobs))
		ids = lo.Map(jobs, func(item models.Job, _ int) string {
			return item.ID
		})
		require.ElementsMatch(t, []string{"110", "120", "130", "140", "150", "160", "170"}, ids)
	})

	s.T().Run("everything", func(t *testing.T) {
		response, err := s.store.GetJobs(s.ctx, jobstore.JobQuery{
			ReturnAll: true,
		})
		require.NoError(t, err)
		require.Equal(t, 7, len(response.Jobs))
	})

	s.T().Run("everything offset", func(t *testing.T) {
		response, err := s.store.GetJobs(s.ctx, jobstore.JobQuery{
			ReturnAll: true,
			Offset:    1,
		})
		require.NoError(t, err)
		require.Equal(t, 6, len(response.Jobs))
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
		require.Equal(t, 6, len(response.Jobs))
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
		[]string{"sh", "-c", "echo hello"})
	job.Labels = map[string]string{"tag": "value"}
	job.ID = "deleteme"
	job.Name = fmt.Sprintf("deleteme-%d", time.Now().UnixNano())
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
	execution.Job = nil
	s.Require().NoError(s.store.CreateJob(s.ctx, *job))
	s.Require().NoError(s.store.CreateExecution(s.ctx, *execution))

	// Ensure that the execution is created
	exec, err := s.store.GetExecutions(s.ctx, jobstore.GetExecutionsOptions{
		JobID:          job.ID,
		AllJobVersions: true,
	})
	s.Require().NoError(err)
	s.Require().Equal(1, len(exec))
	s.Require().Nil(exec[0].Job)

	// Ensure that the execution is created and the job is included
	exec, err = s.store.GetExecutions(s.ctx, jobstore.GetExecutionsOptions{
		JobID:      job.ID,
		JobVersion: 1,
		IncludeJob: true,
	})
	s.Require().NoError(err)
	s.Require().Equal(1, len(exec))
	s.Require().NotNil(exec[0].Job)
	s.Require().Equal(job.ID, exec[0].Job.ID)
}

func (s *BoltJobstoreTestSuite) TestGetExecutions() {
	s.Run("Get execution for existing job", func() {
		state, err := s.store.GetExecutions(s.ctx, jobstore.GetExecutionsOptions{
			JobID: "110",
		})
		s.Require().NoError(err)
		s.NotNil(state)
		s.Equal(1, len(state))
	})

	s.Run("Get execution with included job", func() {
		state, err := s.store.GetExecutions(s.ctx, jobstore.GetExecutionsOptions{
			JobID:      "110",
			IncludeJob: true,
		})
		s.Require().NoError(err)
		s.NotNil(state)
		s.Equal(len(state), 1)
		s.NotNil(state[0].Job)
		s.Equal("110", state[0].Job.ID)
	})

	s.Run("Error on non-existent job ID", func() {
		state, err := s.store.GetExecutions(s.ctx, jobstore.GetExecutionsOptions{
			JobID: "100",
		})
		s.Require().Error(err)
		s.Require().True(bacerrors.IsError(err))
		s.Require().Nil(state)
	})

	s.Run("Find executions using prefix job ID", func() {
		state, err := s.store.GetExecutions(s.ctx, jobstore.GetExecutionsOptions{
			JobID: "11",
		})
		s.Require().NoError(err)
		s.NotNil(state)
		s.Require().Equal("110", state[0].JobID)
	})

	s.Run("Error on ambiguous job ID prefix", func() {
		state, err := s.store.GetExecutions(s.ctx, jobstore.GetExecutionsOptions{
			JobID: "1",
		})
		s.Require().Error(err)
		s.Require().True(bacerrors.IsError(err))
		s.Require().Nil(state)
	})

	s.Run("Created At Ascending Order Sort", func() {
		state, err := s.store.GetExecutions(s.ctx, jobstore.GetExecutionsOptions{
			JobID:   "160",
			OrderBy: "created_at",
		})
		s.Require().NoError(err)
		s.NotNil(state)
		s.Equal(2, len(state))
		s.Equal(state[0].GetCreateTime().Before(state[1].GetCreateTime()), true)
	})

	s.Run("Created At Descending Order Sort", func() {
		state, err := s.store.GetExecutions(s.ctx, jobstore.GetExecutionsOptions{
			JobID:   "160",
			OrderBy: "created_at",
			Reverse: true,
		})
		s.Require().NoError(err)
		s.NotNil(state)
		s.Equal(2, len(state))
		s.Equal(state[0].GetCreateTime().After(state[1].GetCreateTime()), true)
	})

	s.Run("Create Time Backward Compatibility Ascending", func() {
		state, err := s.store.GetExecutions(s.ctx, jobstore.GetExecutionsOptions{
			JobID:   "160",
			OrderBy: "create_time",
		})
		s.Require().NoError(err)
		s.NotNil(state)
		s.Equal(2, len(state))
		s.Equal(state[0].GetCreateTime().Before(state[1].GetCreateTime()), true)
	})

	s.Run("Create Time Backward Compatibility Descending", func() {
		state, err := s.store.GetExecutions(s.ctx, jobstore.GetExecutionsOptions{
			JobID:   "160",
			OrderBy: "create_time",
			Reverse: true,
		})
		s.Require().NoError(err)
		s.NotNil(state)
		s.Equal(2, len(state))
		s.Equal(state[0].GetCreateTime().After(state[1].GetCreateTime()), true)
	})

	s.Run("Default Sort is Created At Ascending", func() {
		state, err := s.store.GetExecutions(s.ctx, jobstore.GetExecutionsOptions{
			JobID: "160",
		})
		s.Require().NoError(err)
		s.NotNil(state)
		s.Equal(2, len(state))
		s.Equal(state[0].GetCreateTime().Before(state[1].GetCreateTime()), true)
	})

	s.Run("Modified At Ascending Order Sort", func() {
		state, err := s.store.GetExecutions(s.ctx, jobstore.GetExecutionsOptions{
			JobID:   "160",
			OrderBy: "modified_at",
		})
		s.Require().NoError(err)
		s.NotNil(state)
		s.Equal(2, len(state))
		s.Equal(state[0].GetModifyTime().Before(state[1].GetModifyTime()), true)
	})

	s.Run("Modified At Descending Order Sort", func() {
		state, err := s.store.GetExecutions(s.ctx, jobstore.GetExecutionsOptions{
			JobID:   "160",
			OrderBy: "modified_at",
			Reverse: true,
		})
		s.Require().NoError(err)
		s.NotNil(state)
		s.Equal(2, len(state))
		s.Equal(state[0].GetModifyTime().After(state[1].GetModifyTime()), true)
	})

	s.Run("Modify Time Backward Compatibility Ascending", func() {
		state, err := s.store.GetExecutions(s.ctx, jobstore.GetExecutionsOptions{
			JobID:   "160",
			OrderBy: "modify_time",
		})
		s.Require().NoError(err)
		s.NotNil(state)
		s.Equal(2, len(state))
		s.Equal(state[0].GetModifyTime().Before(state[1].GetModifyTime()), true)
	})

	s.Run("Modify Time Backward Compatibility Descending", func() {
		state, err := s.store.GetExecutions(s.ctx, jobstore.GetExecutionsOptions{
			JobID:   "160",
			OrderBy: "modify_time",
			Reverse: true,
		})
		s.Require().NoError(err)
		s.NotNil(state)
		s.Equal(2, len(state))
		s.Equal(state[0].GetModifyTime().After(state[1].GetModifyTime()), true)
	})

	s.Run("No job version defined, get latest", func() {
		state, err := s.store.GetExecutions(s.ctx, jobstore.GetExecutionsOptions{
			JobID: "170",
		})
		s.Require().NoError(err)
		s.NotNil(state)
		s.Equal(5, len(state))
	})

	s.Run("Get older job version", func() {
		state, err := s.store.GetExecutions(s.ctx, jobstore.GetExecutionsOptions{
			JobID:      "170",
			JobVersion: 1,
		})
		s.Require().NoError(err)
		s.NotNil(state)
		s.Equal(3, len(state))
		s.Equal(uint64(1), state[0].JobVersion)
	})

	s.Run("Get latest job version", func() {
		state, err := s.store.GetExecutions(s.ctx, jobstore.GetExecutionsOptions{
			JobID:      "170",
			JobVersion: 2,
		})
		s.Require().NoError(err)
		s.NotNil(state)
		s.Equal(5, len(state))
		s.Equal(uint64(2), state[0].JobVersion)
	})

	s.Run("Get all job versions", func() {
		state, err := s.store.GetExecutions(s.ctx, jobstore.GetExecutionsOptions{
			JobID:          "170",
			AllJobVersions: true,
		})
		s.Require().NoError(err)
		s.NotNil(state)
		s.Equal(8, len(state))
	})

	s.Run("In progress executions by jobID", func() {
		state, err := s.store.GetExecutions(s.ctx, jobstore.GetExecutionsOptions{
			JobID:          "170",
			InProgressOnly: true,
		})
		s.Require().NoError(err)
		s.NotNil(state)
		s.Equal(2, len(state))
	})

	s.Run("In progress executions by jobID for older job version", func() {
		state, err := s.store.GetExecutions(s.ctx, jobstore.GetExecutionsOptions{
			JobID:          "170",
			InProgressOnly: true,
			JobVersion:     1,
		})
		s.Require().NoError(err)
		s.NotNil(state)
		s.Equal(1, len(state))
		s.Equal(uint64(1), state[0].JobVersion)
	})

	s.Run("In progress executions by jobID across all versions", func() {
		state, err := s.store.GetExecutions(s.ctx, jobstore.GetExecutionsOptions{
			JobID:          "170",
			InProgressOnly: true,
			AllJobVersions: true,
		})
		s.Require().NoError(err)
		s.NotNil(state)
		s.Equal(3, len(state))
	})

	s.Run("By nodeID and jobID", func() {
		state, err := s.store.GetExecutions(s.ctx, jobstore.GetExecutionsOptions{
			JobID:   "170",
			NodeIDs: []string{"node-1"},
		})
		s.Require().NoError(err)
		s.NotNil(state)
		s.Equal(1, len(state))
		s.Equal("node-1", state[0].NodeID)
		s.Equal(uint64(2), state[0].JobVersion)
	})

	s.Run("By nodeID and jobID with job version", func() {
		state, err := s.store.GetExecutions(s.ctx, jobstore.GetExecutionsOptions{
			JobID:      "170",
			NodeIDs:    []string{"node-1"},
			JobVersion: 1,
		})
		s.Require().NoError(err)
		s.NotNil(state)
		s.Equal(1, len(state))
		s.Equal("node-1", state[0].NodeID)
		s.Equal(uint64(1), state[0].JobVersion)
	})

	s.Run("By nodeID and jobID with all versions", func() {
		state, err := s.store.GetExecutions(s.ctx, jobstore.GetExecutionsOptions{
			JobID:          "170",
			NodeIDs:        []string{"node-1"},
			AllJobVersions: true,
		})
		s.Require().NoError(err)
		s.NotNil(state)
		s.Equal(2, len(state))
		s.Equal("node-1", state[0].NodeID)
		s.Equal("node-1", state[1].NodeID)
		s.Equal(uint64(1), state[0].JobVersion)
		s.Equal(uint64(2), state[1].JobVersion)
	})

	s.Run("In progress executions by nodeID and jobID", func() {
		state, err := s.store.GetExecutions(s.ctx, jobstore.GetExecutionsOptions{
			JobID:          "170",
			InProgressOnly: true,
			NodeIDs:        []string{"node-1", "node-2"},
		})
		s.Require().NoError(err)
		s.NotNil(state)
		s.Equal(2, len(state))
	})

	s.Run("In progress executions by nodeID and jobID across all versions", func() {
		state, err := s.store.GetExecutions(s.ctx, jobstore.GetExecutionsOptions{
			JobID:          "170",
			InProgressOnly: true,
			AllJobVersions: true,
			NodeIDs:        []string{"node-1", "node-2"},
		})
		s.Require().NoError(err)
		s.NotNil(state)
		s.Equal(3, len(state))
	})

	s.Run("In progress executions across all jobs", func() {
		state, err := s.store.GetExecutions(s.ctx, jobstore.GetExecutionsOptions{
			InProgressOnly: true,
		})
		s.Require().NoError(err)
		s.NotNil(state)
		s.Equal(5, len(state))
	})

	s.Run("In progress executions across all jobs with all versions", func() {
		state, err := s.store.GetExecutions(s.ctx, jobstore.GetExecutionsOptions{
			InProgressOnly: true,
			AllJobVersions: true,
		})
		s.Require().NoError(err)
		s.NotNil(state)
		s.Equal(6, len(state))
	})

	s.Run("By nodeID across all jobs", func() {
		state, err := s.store.GetExecutions(s.ctx, jobstore.GetExecutionsOptions{
			NodeIDs: []string{"node-1", "node-2"},
		})
		s.Require().NoError(err)
		s.NotNil(state)
		s.Equal(9, len(state))
	})

	s.Run("By nodeID across all jobs and all versions", func() {
		state, err := s.store.GetExecutions(s.ctx, jobstore.GetExecutionsOptions{
			NodeIDs:        []string{"node-1", "node-2"},
			AllJobVersions: true,
		})
		s.Require().NoError(err)
		s.NotNil(state)
		s.Equal(11, len(state))
	})

	s.Run("By nodeID across all jobs with in-progress only", func() {
		state, err := s.store.GetExecutions(s.ctx, jobstore.GetExecutionsOptions{
			NodeIDs:        []string{"node-1", "node-2"},
			InProgressOnly: true,
		})
		s.Require().NoError(err)
		s.NotNil(state)
		s.Equal(5, len(state))
	})

	// test bad combinations
	s.Run("Bad request: no job ID, no node IDs, no in-progress only", func() {
		_, err := s.store.GetExecutions(s.ctx, jobstore.GetExecutionsOptions{})
		s.Require().Error(err)
		s.Require().True(bacerrors.IsError(err), "Expected an error for no job ID, no node IDs, and no in-progress only")
	})

	s.Run("Bad request: job version defined without job ID", func() {
		_, err := s.store.GetExecutions(s.ctx, jobstore.GetExecutionsOptions{
			JobVersion: 1,
		})
		s.Require().Error(err)
		s.Require().True(bacerrors.IsError(err), "Expected an error for job version defined without job ID")
	})

	s.Run("Bad request: job version and all job versions defined", func() {
		_, err := s.store.GetExecutions(s.ctx, jobstore.GetExecutionsOptions{
			JobID:          "170",
			JobVersion:     1,
			AllJobVersions: true,
		})
		s.Require().NoError(err)
	})

}

func (s *BoltJobstoreTestSuite) TestInProgressJobs() {
	infos, err := s.store.GetInProgressJobs(s.ctx, "")
	s.Require().NoError(err)
	s.Require().Equal(5, len(infos))
	s.Require().Equal("130", infos[0].ID)

	infos, err = s.store.GetInProgressJobs(s.ctx, "batch")
	s.Require().NoError(err)
	s.Require().Equal(4, len(infos))
	s.Require().Equal("130", infos[0].ID)

	infos, err = s.store.GetInProgressJobs(s.ctx, "daemon")
	s.Require().NoError(err)
	s.Require().Equal(1, len(infos))
	s.Require().Equal("150", infos[0].ID)
}

func (s *BoltJobstoreTestSuite) TestShortIDs() {
	uuidString := "9308d0d2-d93c-4e22-8a5b-c392e614922e"
	jobNameString1 := "job-1"
	uuidString2 := "9308d0d2-d93c-4e22-8a5b-c392e614922f"
	jobNameString2 := "job-2"
	shortString := "9308d0d2"

	job := makeDockerEngineJob(
		[]string{"sh", "-c", "echo hello"})
	job.ID = uuidString
	job.Name = jobNameString1
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
	job.Name = jobNameString2
	err = s.store.CreateJob(s.ctx, *job)
	s.Require().NoError(err)

	_, err = s.store.GetJob(s.ctx, shortString)
	s.Require().Error(err)
	s.Require().True(bacerrors.IsError(err))
}

func (s *BoltJobstoreTestSuite) TestEvents() {
	// Create test job
	testJob := mock.Job()
	testJob.ID = "10"
	testJob.Namespace = "110"
	s.Require().NoError(s.store.CreateJob(s.ctx, *testJob))

	// Get sequence number after setup to ignore setup events
	lastSeqNum, err := s.store.GetEventStore().GetLatestEventNum(s.ctx)
	s.Require().NoError(err)

	s.Run("execution events", func() {
		// Create execution
		s.clock.Add(1 * time.Second)
		testExec := *mock.ExecutionForJob(testJob)
		testExec.ComputeState.StateType = models.ExecutionStateNew
		s.Require().NoError(s.store.CreateExecution(s.ctx, testExec))

		// Verify creation event
		event := s.getLastEvent(lastSeqNum, jobstore.EventObjectExecutionUpsert)
		s.verifyExecutionEvent(event, watcher.OperationCreate, testExec.ID, models.ExecutionStateNew, models.ExecutionStateUndefined)
		lastSeqNum = event.SeqNum

		// Test multiple events in execution history
		s.clock.Add(1 * time.Second)
		events := []models.Event{
			*models.NewEvent("test1").WithMessage("message1"),
			*models.NewEvent("test2").WithMessage("message2"),
		}
		s.Require().NoError(s.store.AddExecutionHistory(s.ctx, testJob.ID, testExec.JobVersion, testExec.ID, events...))

		// Update execution state
		s.clock.Add(1 * time.Second)
		testExec.ComputeState.StateType = models.ExecutionStateAskForBid
		updateEvent := models.NewEvent("update").WithMessage("state change")
		s.Require().NoError(s.store.UpdateExecution(s.ctx, jobstore.UpdateExecutionRequest{
			ExecutionID: testExec.ID,
			Condition: jobstore.UpdateExecutionCondition{
				ExpectedStates: []models.ExecutionStateType{models.ExecutionStateNew},
			},
			NewValues: testExec,
			Events:    []*models.Event{updateEvent},
		}))

		// Verify update event has events included
		event = s.getLastEvent(lastSeqNum, jobstore.EventObjectExecutionUpsert)
		s.verifyExecutionEvent(event, watcher.OperationUpdate, testExec.ID,
			models.ExecutionStateAskForBid, models.ExecutionStateNew, updateEvent)
		lastSeqNum = event.SeqNum
	})

	s.Run("evaluation events", func() {
		// Create evaluation
		testEval := mock.EvalForJob(testJob)
		s.Require().NoError(s.store.CreateEvaluation(s.ctx, *testEval))

		// Verify creation event
		event := s.getLastEvent(lastSeqNum, jobstore.EventObjectEvaluation)
		s.verifyEvaluationEvent(event, watcher.OperationCreate, testEval)
		lastSeqNum = event.SeqNum

		// Delete evaluation
		s.Require().NoError(s.store.DeleteEvaluation(s.ctx, testEval.ID))

		// Verify deletion event
		event = s.getLastEvent(lastSeqNum, jobstore.EventObjectEvaluation)
		s.verifyEvaluationEvent(event, watcher.OperationDelete, testEval)
	})
}

// Helper methods for event verification
func (s *BoltJobstoreTestSuite) getLastEvent(afterSeqNum uint64, objectType string) watcher.Event {
	response, err := s.store.GetEventStore().GetEvents(s.ctx, watcher.GetEventsRequest{
		EventIterator: watcher.AfterSequenceNumberIterator(afterSeqNum),
		Filter: watcher.EventFilter{
			ObjectTypes: []string{objectType},
		},
	})
	s.Require().NoError(err)
	s.Require().Equal(1, len(response.Events))
	return response.Events[0]
}

func (s *BoltJobstoreTestSuite) verifyExecutionEvent(
	event watcher.Event,
	expectedOp watcher.Operation,
	execID string,
	state models.ExecutionStateType,
	previousState models.ExecutionStateType,
	events ...*models.Event,
) {
	s.Equal(expectedOp, event.Operation)
	s.Equal(jobstore.EventObjectExecutionUpsert, event.ObjectType)

	upsertEvent, ok := event.Object.(models.ExecutionUpsert)
	s.Require().True(ok)
	s.Equal(execID, upsertEvent.Current.ID)
	s.Equal(state, upsertEvent.Current.ComputeState.StateType)

	if !previousState.IsUndefined() {
		s.Require().NotNil(upsertEvent.Previous)
		s.Equal(previousState, upsertEvent.Previous.ComputeState.StateType)
	} else {
		s.Nil(upsertEvent.Previous)
	}

	s.Require().Equal(len(events), len(upsertEvent.Events))
	for i := range events {
		s.Require().Equal(events[i].Message, upsertEvent.Events[i].Message)
	}
}

func (s *BoltJobstoreTestSuite) verifyEvaluationEvent(
	event watcher.Event,
	expectedOp watcher.Operation,
	expectedEval *models.Evaluation,
) {
	s.Equal(expectedOp, event.Operation)
	s.Equal(jobstore.EventObjectEvaluation, event.ObjectType)

	evalObj, ok := event.Object.(models.Evaluation)
	s.Require().True(ok, "expected object to be an evaluation, but got %T", event.Object)
	s.Equal(expectedEval.ID, evalObj.ID)
	s.Equal(expectedEval.JobID, evalObj.JobID)
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

	defer txCtx.Rollback()

	job := mock.Job()
	execution := mock.ExecutionForJob(job)
	evaluation := mock.EvalForJob(job)
	s.Require().NoError(s.store.CreateJob(txCtx, *job))
	s.Require().NoError(s.store.CreateExecution(txCtx, *execution))

	// cancel the context
	cancel()

	// Ensure operation fails with context canceled
	err = s.store.CreateEvaluation(txCtx, *evaluation)
	s.Require().Error(err)
	s.Require().ErrorIs(err, context.Canceled)
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
	tx1, ok := boltdblib.TxFromContext(txCtx1)
	s.Require().True(ok)
	// commit to release the transaction
	s.Require().NoError(txCtx1.Commit())

	// start second transaction, even through tcCtx1
	txCtx2, err := s.store.BeginTx(txCtx1)
	s.Require().NoError(err)
	s.Require().NotNil(txCtx2)
	tx2, ok := boltdblib.TxFromContext(txCtx2)
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
			"Image":      "busybox:1.37.0",
			"Entrypoint": entrypointArray,
		},
	}
	return j
}

func (s *BoltJobstoreTestSuite) TestGetJobVersion() {
	// Test case 1: Get existing job version
	job, err := s.store.GetJobVersion(s.ctx, "110", 1)
	s.NoError(err)
	s.Equal("110", job.ID)
	s.Equal("client1", job.Namespace)
	s.Equal("batch", job.Type)

	// Test case 2: Get non-existent job version
	_, err = s.store.GetJobVersion(s.ctx, "110", 999)
	s.Error(err)
	s.True(bacerrors.IsErrorWithCode(err, bacerrors.NotFoundError))

	// Test case 3: Get version for non-existent job
	_, err = s.store.GetJobVersion(s.ctx, "non-existent", 1)
	s.Error(err)
	s.True(bacerrors.IsErrorWithCode(err, bacerrors.NotFoundError))
}

func (s *BoltJobstoreTestSuite) TestGetJobVersionWithShortID() {
	// Create a job with a long ID
	job := mock.Job()
	job.ID = "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"
	job.Name = "short-id-job"
	job.Namespace = "test"
	job.Type = models.JobTypeBatch
	job.Tasks = []*models.Task{
		{
			Name: "task1",
			Engine: &models.SpecConfig{
				Type: models.EngineDocker,
			},
		},
	}

	// Create the job
	err := s.store.CreateJob(s.ctx, *job)
	s.Require().NoError(err)

	// Add job history
	err = s.store.AddJobHistory(s.ctx, job.ID, job.Version, *models.NewEvent("job-created").WithMessage("Job created"))
	s.Require().NoError(err)

	// Test getting job version with short ID
	shortID := job.ID[:8]
	retrievedJob, err := s.store.GetJobVersion(s.ctx, shortID, 1)
	s.Require().NoError(err)
	s.Require().Equal(job.ID, retrievedJob.ID)
}

func (s *BoltJobstoreTestSuite) TestGetJobVersionWithMultipleVersions() {
	// Create initial job
	job := mock.Job()
	job.ID = "multi-version-job"
	job.Name = "multi-version-job"
	job.Namespace = "test"
	job.Type = models.JobTypeBatch
	job.Tasks = []*models.Task{
		{
			Name: "task1",
			Engine: &models.SpecConfig{
				Type: models.EngineDocker,
			},
		},
	}
	job.Labels = map[string]string{
		"version": "1",
	}

	// Create the job
	err := s.store.CreateJob(s.ctx, *job)
	s.Require().NoError(err)

	// Add job history for version 1
	err = s.store.AddJobHistory(s.ctx, job.ID, job.Version, *models.NewEvent("job-created").WithMessage("Job created"))
	s.Require().NoError(err)

	// Update job to create version 2
	job.Labels["version"] = "2"
	err = s.store.UpdateJob(s.ctx, *job)
	s.Require().NoError(err)

	// Add job history for version 2
	err = s.store.AddJobHistory(s.ctx, job.ID, job.Version, *models.NewEvent("job-updated").WithMessage("Job updated"))
	s.Require().NoError(err)

	// Test getting version 1
	version1, err := s.store.GetJobVersion(s.ctx, job.ID, 1)
	s.Require().NoError(err)
	s.Require().Equal("1", version1.Labels["version"])

	// Test getting version 2
	version2, err := s.store.GetJobVersion(s.ctx, job.ID, 2)
	s.Require().NoError(err)
	s.Require().Equal("2", version2.Labels["version"])

	// Test getting non-existent version
	_, err = s.store.GetJobVersion(s.ctx, job.ID, 3)
	s.Require().Error(err)
	s.Require().True(bacerrors.IsErrorWithCode(err, bacerrors.NotFoundError))
}

func (s *BoltJobstoreTestSuite) TestGetJobVersionWithInvalidVersion() {
	// Test with version 0
	_, err := s.store.GetJobVersion(s.ctx, "110", 0)
	s.Error(err)
	s.True(bacerrors.IsErrorWithCode(err, bacerrors.NotFoundError))

	// Test with negative version
	_, err = s.store.GetJobVersion(s.ctx, "110", ^uint64(0))
	s.Error(err)
	s.True(bacerrors.IsErrorWithCode(err, bacerrors.NotFoundError))
}

func (s *BoltJobstoreTestSuite) TestGetJobVersions() {
	// Create initial job
	job := mock.Job()
	job.ID = "multi-version-job"
	job.Name = "multi-version-job"
	job.Namespace = "test"
	job.Type = models.JobTypeBatch
	job.Tasks = []*models.Task{
		{
			Name: "task1",
			Engine: &models.SpecConfig{
				Type: models.EngineDocker,
			},
		},
	}
	job.Labels = map[string]string{
		"version": "1",
	}

	// Create the job
	err := s.store.CreateJob(s.ctx, *job)
	s.Require().NoError(err)

	// Add job history for version 1
	err = s.store.AddJobHistory(s.ctx, job.ID, job.Version, *models.NewEvent("job-created").WithMessage("Job created"))
	s.Require().NoError(err)

	// Update job to create version 2
	job.Labels["version"] = "2"
	err = s.store.UpdateJob(s.ctx, *job)
	s.Require().NoError(err)

	// Add job history for version 2
	err = s.store.AddJobHistory(s.ctx, job.ID, job.Version, *models.NewEvent("job-updated").WithMessage("Job updated"))
	s.Require().NoError(err)

	// Get all versions
	versions, err := s.store.GetJobVersions(s.ctx, job.ID)
	s.Require().NoError(err)
	s.Require().Len(versions, 2)
	s.Require().Equal("1", versions[0].Labels["version"])
	s.Require().Equal("2", versions[1].Labels["version"])
}

func (s *BoltJobstoreTestSuite) TestGetJobVersionsWithShortID() {
	// Create a job with a long ID
	job := mock.Job()
	job.ID = "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"
	job.Name = "short-id-job"
	job.Namespace = "test"
	job.Type = models.JobTypeBatch
	job.Tasks = []*models.Task{
		{
			Name: "task1",
			Engine: &models.SpecConfig{
				Type: models.EngineDocker,
			},
		},
	}

	// Create the job
	err := s.store.CreateJob(s.ctx, *job)
	s.Require().NoError(err)

	// Add job history
	err = s.store.AddJobHistory(s.ctx, job.ID, job.Version, *models.NewEvent("job-created").WithMessage("Job created"))
	s.Require().NoError(err)

	// Test getting job versions with short ID
	shortID := job.ID[:8]
	versions, err := s.store.GetJobVersions(s.ctx, shortID)
	s.Require().NoError(err)
	s.Require().Len(versions, 1)
	s.Require().Equal(job.ID, versions[0].ID)
}

func (s *BoltJobstoreTestSuite) TestGetJobVersionsNonExistentJob() {
	// Test getting versions for non-existent job
	versions, err := s.store.GetJobVersions(s.ctx, "non-existent-job")
	s.Require().Error(err)
	s.Require().True(bacerrors.IsErrorWithCode(err, bacerrors.NotFoundError))
	s.Require().Empty(versions)
}

func (s *BoltJobstoreTestSuite) TestGetJobVersionsWithMultipleUpdates() {
	// Create initial job
	job := mock.Job()
	job.ID = "multi-update-job"
	job.Name = "multi-update-job"
	job.Namespace = "test"
	job.Type = models.JobTypeBatch
	job.Tasks = []*models.Task{
		{
			Name: "task1",
			Engine: &models.SpecConfig{
				Type: models.EngineDocker,
			},
		},
	}
	job.Labels = map[string]string{
		"version": "1",
	}

	// Create the job
	err := s.store.CreateJob(s.ctx, *job)
	s.Require().NoError(err)

	// Add job history for version 1
	err = s.store.AddJobHistory(s.ctx, job.ID, job.Version, *models.NewEvent("job-created").WithMessage("Job created"))
	s.Require().NoError(err)

	// Update job multiple times to create more versions
	for i := 2; i <= 5; i++ {
		job.Labels["version"] = fmt.Sprintf("%d", i)
		err = s.store.UpdateJob(s.ctx, *job)
		s.Require().NoError(err)

		// Add job history for each version
		err = s.store.AddJobHistory(s.ctx, job.ID, job.Version, *models.NewEvent("job-updated").WithMessage(fmt.Sprintf("Job updated to version %d", i)))
		s.Require().NoError(err)
	}

	// Get all versions
	versions, err := s.store.GetJobVersions(s.ctx, job.ID)
	s.Require().NoError(err)
	s.Require().Len(versions, 5)

	// Verify versions are in correct order and have correct labels
	for i, version := range versions {
		s.Require().Equal(fmt.Sprintf("%d", i+1), version.Labels["version"])
	}
}

func (s *BoltJobstoreTestSuite) TestGetLatestJobVersion() {
	// Create initial job
	job := mock.Job()
	job.ID = "latest-version-job"
	job.Name = "latest-version-job"
	job.Namespace = "test"
	job.Type = models.JobTypeBatch
	job.Tasks = []*models.Task{
		{
			Name: "task1",
			Engine: &models.SpecConfig{
				Type: models.EngineDocker,
			},
		},
	}

	// Create the job
	err := s.store.CreateJob(s.ctx, *job)
	s.Require().NoError(err)

	// Add job history for version 1
	err = s.store.AddJobHistory(s.ctx, job.ID, job.Version, *models.NewEvent("job-created").WithMessage("Job created"))
	s.Require().NoError(err)

	// Test getting latest version (should be 1) by getting all versions
	versions, err := s.store.GetJobVersions(s.ctx, job.ID)
	s.Require().NoError(err)
	s.Require().Len(versions, 1)
	s.Require().Equal(uint64(1), versions[0].Version)
}

func (s *BoltJobstoreTestSuite) TestGetLatestJobVersionWithShortID() {
	// Create a job with a long ID
	job := mock.Job()
	job.ID = "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"
	job.Name = "latest-short-id-job"
	job.Namespace = "test"
	job.Type = models.JobTypeBatch
	job.Tasks = []*models.Task{
		{
			Name: "task1",
			Engine: &models.SpecConfig{
				Type: models.EngineDocker,
			},
		},
	}

	// Create the job
	err := s.store.CreateJob(s.ctx, *job)
	s.Require().NoError(err)

	// Add job history
	err = s.store.AddJobHistory(s.ctx, job.ID, job.Version, *models.NewEvent("job-created").WithMessage("Job created"))
	s.Require().NoError(err)

	// Test getting latest version with short ID by getting all versions
	shortID := job.ID[:8]
	versions, err := s.store.GetJobVersions(s.ctx, shortID)
	s.Require().NoError(err)
	s.Require().Len(versions, 1)
	s.Require().Equal(uint64(1), versions[0].Version)
	s.Require().Equal(job.ID, versions[0].ID)
}

func (s *BoltJobstoreTestSuite) TestGetLatestJobVersionNonExistentJob() {
	// Test getting latest version for non-existent job by trying to get versions
	versions, err := s.store.GetJobVersions(s.ctx, "non-existent-job")
	s.Require().Error(err)
	s.Require().True(bacerrors.IsErrorWithCode(err, bacerrors.NotFoundError))
	s.Require().Empty(versions)
}

func (s *BoltJobstoreTestSuite) TestGetLatestJobVersionAfterUpdates() {
	// Create initial job
	job := mock.Job()
	job.ID = "latest-after-updates-job"
	job.Name = "latest-after-updates-job"
	job.Namespace = "test"
	job.Type = models.JobTypeBatch
	job.Tasks = []*models.Task{
		{
			Name: "task1",
			Engine: &models.SpecConfig{
				Type: models.EngineDocker,
			},
		},
	}
	job.Labels = map[string]string{
		"version": "1",
	}

	// Create the job
	err := s.store.CreateJob(s.ctx, *job)
	s.Require().NoError(err)

	// Add job history for version 1
	err = s.store.AddJobHistory(s.ctx, job.ID, job.Version, *models.NewEvent("job-created").WithMessage("Job created"))
	s.Require().NoError(err)

	// Update job multiple times to create more versions
	for i := 2; i <= 5; i++ {
		job.Labels["version"] = fmt.Sprintf("%d", i)
		err = s.store.UpdateJob(s.ctx, *job)
		s.Require().NoError(err)

		// Add job history for each version
		err = s.store.AddJobHistory(s.ctx, job.ID, job.Version, *models.NewEvent("job-updated").WithMessage(fmt.Sprintf("Job updated to version %d", i)))
		s.Require().NoError(err)
	}

	// Test that we can get all versions (which indirectly tests getLatestJobVersion)
	versions, err := s.store.GetJobVersions(s.ctx, job.ID)
	s.Require().NoError(err)
	s.Require().Len(versions, 5)

	// Verify the latest version is 5
	latestVersion := versions[len(versions)-1]
	s.Require().Equal(uint64(5), latestVersion.Version)
	s.Require().Equal("5", latestVersion.Labels["version"])
}

func (s *BoltJobstoreTestSuite) TestGetJobByIDOrNameWithID() {
	// Create a job
	job := mock.Job()
	job.ID = "test-job-by-id"
	job.Name = "test-job-by-id"
	job.Namespace = "test"
	job.Type = models.JobTypeBatch
	job.Tasks = []*models.Task{
		{
			Name: "task1",
			Engine: &models.SpecConfig{
				Type: models.EngineDocker,
			},
		},
	}

	// Create the job
	err := s.store.CreateJob(s.ctx, *job)
	s.Require().NoError(err)

	// Add job history
	err = s.store.AddJobHistory(s.ctx, job.ID, job.Version, *models.NewEvent("job-created").WithMessage("Job created"))
	s.Require().NoError(err)

	// Test getting job by ID
	retrievedJob, err := s.store.GetJobByIDOrName(s.ctx, job.ID, job.Namespace)
	s.Require().NoError(err)
	s.Require().Equal(job.ID, retrievedJob.ID)
	s.Require().Equal(job.Name, retrievedJob.Name)
	s.Require().Equal(job.Namespace, retrievedJob.Namespace)
}

func (s *BoltJobstoreTestSuite) TestGetJobByIDOrNameWithName() {
	// Create a job
	job := mock.Job()
	job.ID = "test-job-by-name-id"
	job.Name = "test-job-by-name"
	job.Namespace = "test"
	job.Type = models.JobTypeBatch
	job.Tasks = []*models.Task{
		{
			Name: "task1",
			Engine: &models.SpecConfig{
				Type: models.EngineDocker,
			},
		},
	}

	// Create the job
	err := s.store.CreateJob(s.ctx, *job)
	s.Require().NoError(err)

	// Add job history
	err = s.store.AddJobHistory(s.ctx, job.ID, job.Version, *models.NewEvent("job-created").WithMessage("Job created"))
	s.Require().NoError(err)

	// Test getting job by name
	retrievedJob, err := s.store.GetJobByIDOrName(s.ctx, job.Name, job.Namespace)
	s.Require().NoError(err)
	s.Require().Equal(job.ID, retrievedJob.ID)
	s.Require().Equal(job.Name, retrievedJob.Name)
	s.Require().Equal(job.Namespace, retrievedJob.Namespace)
}

func (s *BoltJobstoreTestSuite) TestGetJobByIDOrNameWithShortID() {
	// Create a job with a long ID
	job := mock.Job()
	job.ID = "1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef"
	job.Name = "test-job-short-id"
	job.Namespace = "test"
	job.Type = models.JobTypeBatch
	job.Tasks = []*models.Task{
		{
			Name: "task1",
			Engine: &models.SpecConfig{
				Type: models.EngineDocker,
			},
		},
	}

	// Create the job
	err := s.store.CreateJob(s.ctx, *job)
	s.Require().NoError(err)

	// Add job history
	err = s.store.AddJobHistory(s.ctx, job.ID, job.Version, *models.NewEvent("job-created").WithMessage("Job created"))
	s.Require().NoError(err)

	// Test getting job by short ID
	shortID := job.ID[:8]
	retrievedJob, err := s.store.GetJobByIDOrName(s.ctx, shortID, job.Namespace)
	s.Require().NoError(err)
	s.Require().Equal(job.ID, retrievedJob.ID)
	s.Require().Equal(job.Name, retrievedJob.Name)
	s.Require().Equal(job.Namespace, retrievedJob.Namespace)
}

func (s *BoltJobstoreTestSuite) TestGetJobByIDOrNameWithDifferentNamespace() {
	// Create a job in a specific namespace
	job := mock.Job()
	job.ID = "test-job-namespace"
	job.Name = "test-job-namespace"
	job.Namespace = "production"
	job.Type = models.JobTypeBatch
	job.Tasks = []*models.Task{
		{
			Name: "task1",
			Engine: &models.SpecConfig{
				Type: models.EngineDocker,
			},
		},
	}

	// Create the job
	err := s.store.CreateJob(s.ctx, *job)
	s.Require().NoError(err)

	// Add job history
	err = s.store.AddJobHistory(s.ctx, job.ID, job.Version, *models.NewEvent("job-created").WithMessage("Job created"))
	s.Require().NoError(err)

	// Test getting job by name in the correct namespace
	retrievedJob, err := s.store.GetJobByIDOrName(s.ctx, job.Name, "production")
	s.Require().NoError(err)
	s.Require().Equal(job.ID, retrievedJob.ID)
	s.Require().Equal(job.Name, retrievedJob.Name)
	s.Require().Equal("production", retrievedJob.Namespace)

	// Test getting job by ID (should work regardless of namespace)
	retrievedJob, err = s.store.GetJobByIDOrName(s.ctx, job.ID, "different-namespace")
	s.Require().NoError(err)
	s.Require().Equal(job.ID, retrievedJob.ID)
	s.Require().Equal("production", retrievedJob.Namespace)
}

func (s *BoltJobstoreTestSuite) TestGetJobByIDOrNameNotFound() {
	// Test getting non-existent job by ID
	_, err := s.store.GetJobByIDOrName(s.ctx, "non-existent-id", "test")
	s.Require().Error(err)
	s.Require().True(bacerrors.IsErrorWithCode(err, bacerrors.NotFoundError))

	// Test getting non-existent job by name
	_, err = s.store.GetJobByIDOrName(s.ctx, "non-existent-name", "test")
	s.Require().Error(err)
	s.Require().True(bacerrors.IsErrorWithCode(err, bacerrors.NotFoundError))
}

func (s *BoltJobstoreTestSuite) TestGetJobByIDOrNameNameInWrongNamespace() {
	// Create a job in one namespace
	job := mock.Job()
	job.ID = "test-job-wrong-namespace"
	job.Name = "test-job-wrong-namespace"
	job.Namespace = "development"
	job.Type = models.JobTypeBatch
	job.Tasks = []*models.Task{
		{
			Name: "task1",
			Engine: &models.SpecConfig{
				Type: models.EngineDocker,
			},
		},
	}

	// Create the job
	err := s.store.CreateJob(s.ctx, *job)
	s.Require().NoError(err)

	// Add job history
	err = s.store.AddJobHistory(s.ctx, job.ID, job.Version, *models.NewEvent("job-created").WithMessage("Job created"))
	s.Require().NoError(err)

	// Test getting job by name in wrong namespace (should fall back to ID lookup and succeed)
	retrievedJob, err := s.store.GetJobByIDOrName(s.ctx, job.Name, "production")
	s.Require().NoError(err)
	s.Require().Equal(job.ID, retrievedJob.ID)
	s.Require().Equal("development", retrievedJob.Namespace)
}

func (s *BoltJobstoreTestSuite) TestGetJobByIDOrNamePrefersByName() {
	// Create two jobs: one with ID matching another's name
	job1 := mock.Job()
	job1.ID = "job-1-id"
	job1.Name = "unique-job-1"
	job1.Namespace = "test"
	job1.Type = models.JobTypeBatch
	job1.Tasks = []*models.Task{
		{
			Name: "task1",
			Engine: &models.SpecConfig{
				Type: models.EngineDocker,
			},
		},
	}
	job1.Labels = map[string]string{"type": "job1"}

	job2 := mock.Job()
	job2.ID = "unique-job-1" // This ID matches job1's name
	job2.Name = "unique-job-2"
	job2.Namespace = "test"
	job2.Type = models.JobTypeBatch
	job2.Tasks = []*models.Task{
		{
			Name: "task1",
			Engine: &models.SpecConfig{
				Type: models.EngineDocker,
			},
		},
	}
	job2.Labels = map[string]string{"type": "job2"}

	// Create both jobs
	err := s.store.CreateJob(s.ctx, *job1)
	s.Require().NoError(err)
	err = s.store.CreateJob(s.ctx, *job2)
	s.Require().NoError(err)

	// Add job history for both
	err = s.store.AddJobHistory(s.ctx, job1.ID, job1.Version, *models.NewEvent("job-created").WithMessage("Job 1 created"))
	s.Require().NoError(err)
	err = s.store.AddJobHistory(s.ctx, job2.ID, job2.Version, *models.NewEvent("job-created").WithMessage("Job 2 created"))
	s.Require().NoError(err)

	// Test that GetJobByIDOrName prefers name lookup over ID lookup
	// When searching for "unique-job-1", it should find job1 by name, not job2 by ID
	retrievedJob, err := s.store.GetJobByIDOrName(s.ctx, "unique-job-1", "test")
	s.Require().NoError(err)
	s.Require().Equal(job1.ID, retrievedJob.ID)            // Should be job1's ID
	s.Require().Equal("unique-job-1", retrievedJob.Name)   // Should be job1's name
	s.Require().Equal("job1", retrievedJob.Labels["type"]) // Should be job1's label
}

func (s *BoltJobstoreTestSuite) TestGetJobByName() {
	// Create a job
	job := mock.Job()
	job.ID = "test-job-by-name-basic"
	job.Name = "basic-job-name"
	job.Namespace = "test"
	job.Type = models.JobTypeBatch
	job.Tasks = []*models.Task{
		{
			Name: "task1",
			Engine: &models.SpecConfig{
				Type: models.EngineDocker,
			},
		},
	}

	// Create the job
	err := s.store.CreateJob(s.ctx, *job)
	s.Require().NoError(err)

	// Add job history
	err = s.store.AddJobHistory(s.ctx, job.ID, job.Version, *models.NewEvent("job-created").WithMessage("Job created"))
	s.Require().NoError(err)

	// Test getting job by name
	retrievedJob, err := s.store.GetJobByName(s.ctx, job.Name, job.Namespace)
	s.Require().NoError(err)
	s.Require().Equal(job.ID, retrievedJob.ID)
	s.Require().Equal(job.Name, retrievedJob.Name)
	s.Require().Equal(job.Namespace, retrievedJob.Namespace)
}

func (s *BoltJobstoreTestSuite) TestGetJobByNameWithDefaultNamespace() {
	// Create a job in default namespace
	job := mock.Job()
	job.ID = "test-job-default-namespace"
	job.Name = "default-namespace-job"
	job.Namespace = models.DefaultNamespace
	job.Type = models.JobTypeBatch
	job.Tasks = []*models.Task{
		{
			Name: "task1",
			Engine: &models.SpecConfig{
				Type: models.EngineDocker,
			},
		},
	}

	// Create the job
	err := s.store.CreateJob(s.ctx, *job)
	s.Require().NoError(err)

	// Add job history
	err = s.store.AddJobHistory(s.ctx, job.ID, job.Version, *models.NewEvent("job-created").WithMessage("Job created"))
	s.Require().NoError(err)

	// Test getting job by name with empty namespace (should default to DefaultNamespace)
	retrievedJob, err := s.store.GetJobByName(s.ctx, job.Name, "")
	s.Require().NoError(err)
	s.Require().Equal(job.ID, retrievedJob.ID)
	s.Require().Equal(job.Name, retrievedJob.Name)
	s.Require().Equal(models.DefaultNamespace, retrievedJob.Namespace)

	// Test getting job by name with explicit default namespace
	retrievedJob, err = s.store.GetJobByName(s.ctx, job.Name, models.DefaultNamespace)
	s.Require().NoError(err)
	s.Require().Equal(job.ID, retrievedJob.ID)
	s.Require().Equal(job.Name, retrievedJob.Name)
	s.Require().Equal(models.DefaultNamespace, retrievedJob.Namespace)
}

func (s *BoltJobstoreTestSuite) TestGetJobByNameWithCustomNamespace() {
	// Create a job in a custom namespace
	job := mock.Job()
	job.ID = "test-job-custom-namespace"
	job.Name = "custom-namespace-job"
	job.Namespace = "production"
	job.Type = models.JobTypeBatch
	job.Tasks = []*models.Task{
		{
			Name: "task1",
			Engine: &models.SpecConfig{
				Type: models.EngineDocker,
			},
		},
	}

	// Create the job
	err := s.store.CreateJob(s.ctx, *job)
	s.Require().NoError(err)

	// Add job history
	err = s.store.AddJobHistory(s.ctx, job.ID, job.Version, *models.NewEvent("job-created").WithMessage("Job created"))
	s.Require().NoError(err)

	// Test getting job by name in the correct namespace
	retrievedJob, err := s.store.GetJobByName(s.ctx, job.Name, "production")
	s.Require().NoError(err)
	s.Require().Equal(job.ID, retrievedJob.ID)
	s.Require().Equal(job.Name, retrievedJob.Name)
	s.Require().Equal("production", retrievedJob.Namespace)
}

func (s *BoltJobstoreTestSuite) TestGetJobByNameNotFound() {
	// Test getting non-existent job by name
	_, err := s.store.GetJobByName(s.ctx, "non-existent-job", "test")
	s.Require().Error(err)
	s.Require().True(bacerrors.IsErrorWithCode(err, bacerrors.NotFoundError))
}

func (s *BoltJobstoreTestSuite) TestGetJobByNameWrongNamespace() {
	// Create a job in one namespace
	job := mock.Job()
	job.ID = "test-job-wrong-namespace-lookup"
	job.Name = "namespace-specific-job"
	job.Namespace = "development"
	job.Type = models.JobTypeBatch
	job.Tasks = []*models.Task{
		{
			Name: "task1",
			Engine: &models.SpecConfig{
				Type: models.EngineDocker,
			},
		},
	}

	// Create the job
	err := s.store.CreateJob(s.ctx, *job)
	s.Require().NoError(err)

	// Add job history
	err = s.store.AddJobHistory(s.ctx, job.ID, job.Version, *models.NewEvent("job-created").WithMessage("Job created"))
	s.Require().NoError(err)

	// Test getting job by name in wrong namespace (should fail)
	_, err = s.store.GetJobByName(s.ctx, job.Name, "production")
	s.Require().Error(err)
	s.Require().True(bacerrors.IsErrorWithCode(err, bacerrors.NotFoundError))
}

func (s *BoltJobstoreTestSuite) TestGetJobByNameMultipleJobsSameName() {
	// Create jobs with same name in different namespaces
	job1 := mock.Job()
	job1.ID = "test-job-1-same-name"
	job1.Name = "same-job-name"
	job1.Namespace = "namespace1"
	job1.Type = models.JobTypeBatch
	job1.Tasks = []*models.Task{
		{
			Name: "task1",
			Engine: &models.SpecConfig{
				Type: models.EngineDocker,
			},
		},
	}
	job1.Labels = map[string]string{"env": "namespace1"}

	job2 := mock.Job()
	job2.ID = "test-job-2-same-name"
	job2.Name = "same-job-name"
	job2.Namespace = "namespace2"
	job2.Type = models.JobTypeBatch
	job2.Tasks = []*models.Task{
		{
			Name: "task1",
			Engine: &models.SpecConfig{
				Type: models.EngineDocker,
			},
		},
	}
	job2.Labels = map[string]string{"env": "namespace2"}

	// Create both jobs
	err := s.store.CreateJob(s.ctx, *job1)
	s.Require().NoError(err)
	err = s.store.CreateJob(s.ctx, *job2)
	s.Require().NoError(err)

	// Add job history for both
	err = s.store.AddJobHistory(s.ctx, job1.ID, job1.Version, *models.NewEvent("job-created").WithMessage("Job 1 created"))
	s.Require().NoError(err)
	err = s.store.AddJobHistory(s.ctx, job2.ID, job2.Version, *models.NewEvent("job-created").WithMessage("Job 2 created"))
	s.Require().NoError(err)

	// Test getting job from namespace1
	retrievedJob1, err := s.store.GetJobByName(s.ctx, "same-job-name", "namespace1")
	s.Require().NoError(err)
	s.Require().Equal(job1.ID, retrievedJob1.ID)
	s.Require().Equal("namespace1", retrievedJob1.Namespace)
	s.Require().Equal("namespace1", retrievedJob1.Labels["env"])

	// Test getting job from namespace2
	retrievedJob2, err := s.store.GetJobByName(s.ctx, "same-job-name", "namespace2")
	s.Require().NoError(err)
	s.Require().Equal(job2.ID, retrievedJob2.ID)
	s.Require().Equal("namespace2", retrievedJob2.Namespace)
	s.Require().Equal("namespace2", retrievedJob2.Labels["env"])
}

func (s *BoltJobstoreTestSuite) TestGetJobByNameCaseSensitive() {
	// Create a job with a specific case name
	job := mock.Job()
	job.ID = "test-job-case-sensitive"
	job.Name = "CaseSensitiveJobName"
	job.Namespace = "test"
	job.Type = models.JobTypeBatch
	job.Tasks = []*models.Task{
		{
			Name: "task1",
			Engine: &models.SpecConfig{
				Type: models.EngineDocker,
			},
		},
	}

	// Create the job
	err := s.store.CreateJob(s.ctx, *job)
	s.Require().NoError(err)

	// Add job history
	err = s.store.AddJobHistory(s.ctx, job.ID, job.Version, *models.NewEvent("job-created").WithMessage("Job created"))
	s.Require().NoError(err)

	// Test getting job with exact case
	retrievedJob, err := s.store.GetJobByName(s.ctx, "CaseSensitiveJobName", "test")
	s.Require().NoError(err)
	s.Require().Equal(job.ID, retrievedJob.ID)
	s.Require().Equal("CaseSensitiveJobName", retrievedJob.Name)

	// Test getting job with different case (should fail)
	_, err = s.store.GetJobByName(s.ctx, "casesensitivejobname", "test")
	s.Require().Error(err)
	s.Require().True(bacerrors.IsErrorWithCode(err, bacerrors.NotFoundError))

	// Test getting job with different case (should fail)
	_, err = s.store.GetJobByName(s.ctx, "CASESENSITIVEJOBNAME", "test")
	s.Require().Error(err)
	s.Require().True(bacerrors.IsErrorWithCode(err, bacerrors.NotFoundError))
}

func (s *BoltJobstoreTestSuite) TestGetJobByNameSpecialCharacters() {
	// Create a job with special characters in name
	job := mock.Job()
	job.ID = "test-job-special-chars"
	job.Name = "job-with-special_chars.123"
	job.Namespace = "test"
	job.Type = models.JobTypeBatch
	job.Tasks = []*models.Task{
		{
			Name: "task1",
			Engine: &models.SpecConfig{
				Type: models.EngineDocker,
			},
		},
	}

	// Create the job
	err := s.store.CreateJob(s.ctx, *job)
	s.Require().NoError(err)

	// Add job history
	err = s.store.AddJobHistory(s.ctx, job.ID, job.Version, *models.NewEvent("job-created").WithMessage("Job created"))
	s.Require().NoError(err)

	// Test getting job with special characters
	retrievedJob, err := s.store.GetJobByName(s.ctx, "job-with-special_chars.123", "test")
	s.Require().NoError(err)
	s.Require().Equal(job.ID, retrievedJob.ID)
	s.Require().Equal("job-with-special_chars.123", retrievedJob.Name)
	s.Require().Equal("test", retrievedJob.Namespace)
}

func (s *BoltJobstoreTestSuite) TestUpdateJob() {
	// Create an initial job
	job := mock.Job()
	job.ID = "test-update-job-basic"
	job.Name = "update-job-test"
	job.Namespace = "test"
	job.Type = models.JobTypeBatch
	job.Priority = 1
	job.Count = 2
	job.Labels = map[string]string{"env": "test", "version": "1.0"}
	job.Tasks = []*models.Task{
		{
			Name: "task1",
			Engine: &models.SpecConfig{
				Type: models.EngineDocker,
			},
		},
	}

	// Create the job
	err := s.store.CreateJob(s.ctx, *job)
	s.Require().NoError(err)

	// Add job history
	err = s.store.AddJobHistory(s.ctx, job.ID, job.Version, *models.NewEvent("job-created").WithMessage("Job created"))
	s.Require().NoError(err)

	// Prepare update data
	updatedJob := *job
	updatedJob.Priority = 5
	updatedJob.Count = 10
	updatedJob.Labels = map[string]string{"env": "production", "version": "2.0"}
	updatedJob.Meta = map[string]string{"updated": "true"}

	// Update the job
	err = s.store.UpdateJob(s.ctx, updatedJob)
	s.Require().NoError(err)

	// Retrieve the updated job
	retrievedJob, err := s.store.GetJob(s.ctx, job.ID)
	s.Require().NoError(err)

	// Verify updates
	s.Require().Equal(uint64(2), retrievedJob.Version) // Version should increment
	s.Require().Equal(5, retrievedJob.Priority)
	s.Require().Equal(10, retrievedJob.Count)
	s.Require().Equal("production", retrievedJob.Labels["env"])
	s.Require().Equal("2.0", retrievedJob.Labels["version"])
	s.Require().Equal("true", retrievedJob.Meta["updated"])
	s.Require().Equal(models.JobStateTypePending, retrievedJob.State.StateType) // State should reset to pending
	s.Require().True(retrievedJob.ModifyTime > job.ModifyTime)                  // ModifyTime should be updated
}

func (s *BoltJobstoreTestSuite) TestUpdateJobWithoutID() {
	// Create a job without ID
	job := mock.Job()
	job.ID = "" // Empty ID
	job.Name = "no-id-job"

	// Attempt to update job without ID
	err := s.store.UpdateJob(s.ctx, *job)
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "cannot update job without an ID")
}

func (s *BoltJobstoreTestSuite) TestUpdateJobNonExistent() {
	// Try to update a job that doesn't exist
	job := mock.Job()
	job.ID = "non-existent-job-update"
	job.Name = "non-existent"
	job.Tasks = []*models.Task{
		{
			Name: "task1",
			Engine: &models.SpecConfig{
				Type: models.EngineDocker,
			},
		},
	}

	err := s.store.UpdateJob(s.ctx, *job)
	s.Require().Error(err)
	s.Require().True(bacerrors.IsErrorWithCode(err, bacerrors.NotFoundError))
}

func (s *BoltJobstoreTestSuite) TestUpdateJobCannotChangeName() {
	// Create an initial job
	job := mock.Job()
	job.ID = "test-update-job-name-change"
	job.Name = "original-name"
	job.Namespace = "test"
	job.Type = models.JobTypeBatch
	job.Tasks = []*models.Task{
		{
			Name: "task1",
			Engine: &models.SpecConfig{
				Type: models.EngineDocker,
			},
		},
	}

	// Create the job
	err := s.store.CreateJob(s.ctx, *job)
	s.Require().NoError(err)

	// Add job history
	err = s.store.AddJobHistory(s.ctx, job.ID, job.Version, *models.NewEvent("job-created").WithMessage("Job created"))
	s.Require().NoError(err)

	// Try to update with different name
	updatedJob := *job
	updatedJob.Name = "new-name"

	err = s.store.UpdateJob(s.ctx, updatedJob)
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "cannot change job name or namespace during update")
}

func (s *BoltJobstoreTestSuite) TestUpdateJobCannotChangeNamespace() {
	// Create an initial job
	job := mock.Job()
	job.ID = "test-update-job-namespace-change"
	job.Name = "namespace-test-job"
	job.Namespace = "original-namespace"
	job.Type = models.JobTypeBatch
	job.Tasks = []*models.Task{
		{
			Name: "task1",
			Engine: &models.SpecConfig{
				Type: models.EngineDocker,
			},
		},
	}

	// Create the job
	err := s.store.CreateJob(s.ctx, *job)
	s.Require().NoError(err)

	// Add job history
	err = s.store.AddJobHistory(s.ctx, job.ID, job.Version, *models.NewEvent("job-created").WithMessage("Job created"))
	s.Require().NoError(err)

	// Try to update with different namespace
	updatedJob := *job
	updatedJob.Namespace = "new-namespace"

	err = s.store.UpdateJob(s.ctx, updatedJob)
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "cannot change job name or namespace during update")
}

func (s *BoltJobstoreTestSuite) TestUpdateJobVersionHistory() {
	// Create an initial job
	job := mock.Job()
	job.ID = "test-update-job-version-history"
	job.Name = "version-history-job"
	job.Namespace = "test"
	job.Type = models.JobTypeBatch
	job.Priority = 1
	job.Tasks = []*models.Task{
		{
			Name: "task1",
			Engine: &models.SpecConfig{
				Type: models.EngineDocker,
			},
		},
	}

	// Create the job
	err := s.store.CreateJob(s.ctx, *job)
	s.Require().NoError(err)

	// Add job history
	err = s.store.AddJobHistory(s.ctx, job.ID, job.Version, *models.NewEvent("job-created").WithMessage("Job created"))
	s.Require().NoError(err)

	// Get the job after creation to capture the actual modification time set by the database
	createdJob, err := s.store.GetJob(s.ctx, job.ID)
	s.Require().NoError(err)

	// Store original version and modification time
	originalVersion := createdJob.Version
	originalModifyTime := createdJob.ModifyTime

	// Advance the mock clock to ensure different modification time
	s.clock.Add(1 * time.Second)

	// Update the job
	updatedJob := *job
	updatedJob.Priority = 10

	err = s.store.UpdateJob(s.ctx, updatedJob)
	s.Require().NoError(err)

	// Add job history for the update (version incremented by UpdateJob)
	err = s.store.AddJobHistory(s.ctx, job.ID, createdJob.Version+1, *models.NewEvent("job-updated").WithMessage("Job priority updated"))
	s.Require().NoError(err)

	// Verify version history
	versions, err := s.store.GetJobVersions(s.ctx, job.ID)
	s.Require().NoError(err)
	s.Require().Len(versions, 2) // Original + updated version

	// Check original version is preserved
	originalVersionJob, err := s.store.GetJobVersion(s.ctx, job.ID, originalVersion)
	s.Require().NoError(err)
	s.Require().Equal(1, originalVersionJob.Priority)
	s.Require().Equal(originalModifyTime, originalVersionJob.ModifyTime)

	// Check current version
	currentJob, err := s.store.GetJob(s.ctx, job.ID)
	s.Require().NoError(err)
	s.Require().Equal(uint64(2), currentJob.Version)
	s.Require().Equal(10, currentJob.Priority)
	s.Require().True(currentJob.ModifyTime > originalModifyTime)
}

func (s *BoltJobstoreTestSuite) TestUpdateJobMultipleUpdates() {
	// Create an initial job
	job := mock.Job()
	job.ID = "test-update-job-multiple"
	job.Name = "multiple-updates-job"
	job.Namespace = "test"
	job.Type = models.JobTypeBatch
	job.Priority = 1
	job.Count = 1
	job.Tasks = []*models.Task{
		{
			Name: "task1",
			Engine: &models.SpecConfig{
				Type: models.EngineDocker,
			},
		},
	}

	// Create the job
	err := s.store.CreateJob(s.ctx, *job)
	s.Require().NoError(err)

	// Add job history
	err = s.store.AddJobHistory(s.ctx, job.ID, job.Version, *models.NewEvent("job-created").WithMessage("Job created"))
	s.Require().NoError(err)

	// Perform multiple updates
	for i := 1; i <= 3; i++ {
		// Get the current job to ensure we have the latest version
		currentJob, err := s.store.GetJob(s.ctx, job.ID)
		s.Require().NoError(err)

		// Update the job
		updatedJob := currentJob
		updatedJob.Priority = i * 10
		updatedJob.Count = i * 5

		err = s.store.UpdateJob(s.ctx, updatedJob)
		s.Require().NoError(err)

		// Add job history for the update
		err = s.store.AddJobHistory(s.ctx, job.ID, uint64(i+1), *models.NewEvent("job-updated").WithMessage(fmt.Sprintf("Update %d", i)))
		s.Require().NoError(err)
	}

	// Verify final state
	finalJob, err := s.store.GetJob(s.ctx, job.ID)
	s.Require().NoError(err)
	s.Require().Equal(uint64(4), finalJob.Version) // 1 initial + 3 updates
	s.Require().Equal(30, finalJob.Priority)       // 3 * 10
	s.Require().Equal(15, finalJob.Count)          // 3 * 5

	// Verify all versions exist
	versions, err := s.store.GetJobVersions(s.ctx, job.ID)
	s.Require().NoError(err)
	s.Require().Len(versions, 4) // Original + 3 updates
}

func (s *BoltJobstoreTestSuite) TestUpdateJobConstraints() {
	// Create an initial job
	job := mock.Job()
	job.ID = "test-update-job-constraints"
	job.Name = "constraints-job"
	job.Namespace = "test"
	job.Type = models.JobTypeBatch
	job.Tasks = []*models.Task{
		{
			Name: "task1",
			Engine: &models.SpecConfig{
				Type: models.EngineDocker,
			},
		},
	}

	// Create the job
	err := s.store.CreateJob(s.ctx, *job)
	s.Require().NoError(err)

	// Add job history
	err = s.store.AddJobHistory(s.ctx, job.ID, job.Version, *models.NewEvent("job-created").WithMessage("Job created"))
	s.Require().NoError(err)

	// Update with constraints
	updatedJob := *job
	updatedJob.Constraints = []*models.LabelSelectorRequirement{
		{
			Key:      "node-type",
			Operator: selection.In,
			Values:   []string{"compute", "gpu"},
		},
	}

	err = s.store.UpdateJob(s.ctx, updatedJob)
	s.Require().NoError(err)

	// Add job history for the update (version incremented by UpdateJob)
	err = s.store.AddJobHistory(s.ctx, job.ID, job.Version+1, *models.NewEvent("job-updated").WithMessage("Job constraints updated"))
	s.Require().NoError(err)

	// Verify constraints were updated
	retrievedJob, err := s.store.GetJob(s.ctx, job.ID)
	s.Require().NoError(err)
	s.Require().Len(retrievedJob.Constraints, 1)
	s.Require().Equal("node-type", retrievedJob.Constraints[0].Key)
	s.Require().Equal(selection.In, retrievedJob.Constraints[0].Operator)
	s.Require().Equal([]string{"compute", "gpu"}, retrievedJob.Constraints[0].Values)
}

func (s *BoltJobstoreTestSuite) TestUpdateJobTasks() {
	// Create an initial job
	job := mock.Job()
	job.ID = "test-update-job-tasks"
	job.Name = "tasks-job"
	job.Namespace = "test"
	job.Type = models.JobTypeBatch
	job.Tasks = []*models.Task{
		{
			Name: "original-task",
			Engine: &models.SpecConfig{
				Type: models.EngineDocker,
			},
		},
	}

	// Create the job
	err := s.store.CreateJob(s.ctx, *job)
	s.Require().NoError(err)

	// Add job history
	err = s.store.AddJobHistory(s.ctx, job.ID, job.Version, *models.NewEvent("job-created").WithMessage("Job created"))
	s.Require().NoError(err)

	// Update with new tasks
	updatedJob := *job
	updatedJob.Tasks = []*models.Task{
		{
			Name: "updated-task",
			Engine: &models.SpecConfig{
				Type: models.EngineDocker,
			},
		},
		{
			Name: "additional-task",
			Engine: &models.SpecConfig{
				Type: models.EngineWasm,
			},
		},
	}

	err = s.store.UpdateJob(s.ctx, updatedJob)
	s.Require().NoError(err)

	// Add job history for the update (version incremented by UpdateJob)
	err = s.store.AddJobHistory(s.ctx, job.ID, job.Version+1, *models.NewEvent("job-updated").WithMessage("Job tasks updated"))
	s.Require().NoError(err)

	// Verify tasks were updated
	retrievedJob, err := s.store.GetJob(s.ctx, job.ID)
	s.Require().NoError(err)
	s.Require().Len(retrievedJob.Tasks, 2)
	s.Require().Equal("updated-task", retrievedJob.Tasks[0].Name)
	s.Require().Equal("additional-task", retrievedJob.Tasks[1].Name)
	s.Require().Equal(models.EngineDocker, retrievedJob.Tasks[0].Engine.Type)
	s.Require().Equal(models.EngineWasm, retrievedJob.Tasks[1].Engine.Type)
}

func (s *BoltJobstoreTestSuite) TestUpdateJobStatePendingReset() {
	// Create an initial job and change its state
	job := mock.Job()
	job.ID = "test-update-job-state-reset"
	job.Name = "state-reset-job"
	job.Namespace = "test"
	job.Type = models.JobTypeBatch
	job.Tasks = []*models.Task{
		{
			Name: "task1",
			Engine: &models.SpecConfig{
				Type: models.EngineDocker,
			},
		},
	}

	// Create the job
	err := s.store.CreateJob(s.ctx, *job)
	s.Require().NoError(err)

	// Add job history
	err = s.store.AddJobHistory(s.ctx, job.ID, job.Version, *models.NewEvent("job-created").WithMessage("Job created"))
	s.Require().NoError(err)

	// Change job state to running
	err = s.store.UpdateJobState(s.ctx, jobstore.UpdateJobStateRequest{
		JobID:    job.ID,
		NewState: models.JobStateTypeRunning,
		Message:  "Job is running",
	})
	s.Require().NoError(err)

	// Verify state is running
	runningJob, err := s.store.GetJob(s.ctx, job.ID)
	s.Require().NoError(err)
	s.Require().Equal(models.JobStateTypeRunning, runningJob.State.StateType)

	// Update the job (should reset state to pending)
	updatedJob := runningJob
	updatedJob.Priority = 10

	err = s.store.UpdateJob(s.ctx, updatedJob)
	s.Require().NoError(err)

	// Add job history for the update (version incremented by UpdateJob)
	err = s.store.AddJobHistory(s.ctx, job.ID, runningJob.Version+1, *models.NewEvent("job-updated").WithMessage("Job priority updated"))
	s.Require().NoError(err)

	// Verify state was reset to pending
	finalJob, err := s.store.GetJob(s.ctx, job.ID)
	s.Require().NoError(err)
	s.Require().Equal(models.JobStateTypePending, finalJob.State.StateType)
	s.Require().Equal(10, finalJob.Priority)
}

func (s *BoltJobstoreTestSuite) TestUpdateJobInvalidData() {
	// Create an initial job
	job := mock.Job()
	job.ID = "test-update-job-invalid-data"
	job.Name = "invalid-data-job"
	job.Namespace = "test"
	job.Type = models.JobTypeBatch
	job.Tasks = []*models.Task{
		{
			Name: "task1",
			Engine: &models.SpecConfig{
				Type: models.EngineDocker,
			},
		},
	}

	// Create the job
	err := s.store.CreateJob(s.ctx, *job)
	s.Require().NoError(err)

	// Add job history
	err = s.store.AddJobHistory(s.ctx, job.ID, job.Version, *models.NewEvent("job-created").WithMessage("Job created"))
	s.Require().NoError(err)

	// Try to update with invalid data (empty tasks)
	updatedJob := *job
	updatedJob.Tasks = []*models.Task{} // Empty tasks should be invalid

	err = s.store.UpdateJob(s.ctx, updatedJob)
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "missing job tasks")
}

func (s *BoltJobstoreTestSuite) TestGetJobIDByJobName() {
	// Create a job
	job := mock.Job()
	job.ID = "test-job-id-by-name-basic"
	job.Name = "basic-job-name"
	job.Namespace = "test"
	job.Type = models.JobTypeBatch
	job.Tasks = []*models.Task{
		{
			Name: "task1",
			Engine: &models.SpecConfig{
				Type: models.EngineDocker,
			},
		},
	}

	// Create the job
	err := s.store.CreateJob(s.ctx, *job)
	s.Require().NoError(err)

	// Add job history
	err = s.store.AddJobHistory(s.ctx, job.ID, job.Version, *models.NewEvent("job-created").WithMessage("Job created"))
	s.Require().NoError(err)

	// Test getting job ID by job name key
	jobNameKey := fmt.Sprintf("%s:%s", job.Name, job.Namespace)

	// Use a transaction to test the private method
	err = boltdblib.View(s.ctx, s.store.database, func(tx *bolt.Tx) error {
		recorder := s.store.metricRecorder(s.ctx, "test", "test")

		retrievedJobID, err := s.store.getJobIDByJobName(s.ctx, tx, recorder, jobNameKey)
		s.Require().NoError(err)
		s.Require().Equal(job.ID, retrievedJobID)

		return nil
	})
	s.Require().NoError(err)
}

func (s *BoltJobstoreTestSuite) TestGetJobIDByJobNameWithDefaultNamespace() {
	// Create a job with default namespace
	job := mock.Job()
	job.ID = "test-job-id-by-name-default-ns"
	job.Name = "default-namespace-job"
	job.Namespace = models.DefaultNamespace
	job.Type = models.JobTypeBatch
	job.Tasks = []*models.Task{
		{
			Name: "task1",
			Engine: &models.SpecConfig{
				Type: models.EngineDocker,
			},
		},
	}

	// Create the job
	err := s.store.CreateJob(s.ctx, *job)
	s.Require().NoError(err)

	// Add job history
	err = s.store.AddJobHistory(s.ctx, job.ID, job.Version, *models.NewEvent("job-created").WithMessage("Job created"))
	s.Require().NoError(err)

	// Test getting job ID by job name key with default namespace
	jobNameKey := fmt.Sprintf("%s:%s", job.Name, models.DefaultNamespace)

	// Use a transaction to test the private method
	err = boltdblib.View(s.ctx, s.store.database, func(tx *bolt.Tx) error {
		recorder := s.store.metricRecorder(s.ctx, "test", "test")

		retrievedJobID, err := s.store.getJobIDByJobName(s.ctx, tx, recorder, jobNameKey)
		s.Require().NoError(err)
		s.Require().Equal(job.ID, retrievedJobID)

		return nil
	})
	s.Require().NoError(err)
}

func (s *BoltJobstoreTestSuite) TestGetJobIDByJobNameWithCustomNamespace() {
	// Create a job with custom namespace
	job := mock.Job()
	job.ID = "test-job-id-by-name-custom-ns"
	job.Name = "custom-namespace-job"
	job.Namespace = "production"
	job.Type = models.JobTypeBatch
	job.Tasks = []*models.Task{
		{
			Name: "task1",
			Engine: &models.SpecConfig{
				Type: models.EngineDocker,
			},
		},
	}

	// Create the job
	err := s.store.CreateJob(s.ctx, *job)
	s.Require().NoError(err)

	// Add job history
	err = s.store.AddJobHistory(s.ctx, job.ID, job.Version, *models.NewEvent("job-created").WithMessage("Job created"))
	s.Require().NoError(err)

	// Test getting job ID by job name key with custom namespace
	jobNameKey := fmt.Sprintf("%s:%s", job.Name, "production")

	// Use a transaction to test the private method
	err = boltdblib.View(s.ctx, s.store.database, func(tx *bolt.Tx) error {
		recorder := s.store.metricRecorder(s.ctx, "test", "test")

		retrievedJobID, err := s.store.getJobIDByJobName(s.ctx, tx, recorder, jobNameKey)
		s.Require().NoError(err)
		s.Require().Equal(job.ID, retrievedJobID)

		return nil
	})
	s.Require().NoError(err)
}

func (s *BoltJobstoreTestSuite) TestGetJobIDByJobNameNotFound() {
	// Test getting job ID for non-existent job name
	jobNameKey := "non-existent-job:test"

	// Use a transaction to test the private method
	err := boltdblib.View(s.ctx, s.store.database, func(tx *bolt.Tx) error {
		recorder := s.store.metricRecorder(s.ctx, "test", "test")

		_, err := s.store.getJobIDByJobName(s.ctx, tx, recorder, jobNameKey)
		s.Require().Error(err)
		s.Require().True(bacerrors.IsErrorWithCode(err, bacerrors.NotFoundError))

		return nil
	})
	s.Require().NoError(err)
}

func (s *BoltJobstoreTestSuite) TestGetJobIDByJobNameWrongNamespace() {
	// Create a job in one namespace
	job := mock.Job()
	job.ID = "test-job-id-wrong-namespace"
	job.Name = "namespace-test-job"
	job.Namespace = "production"
	job.Type = models.JobTypeBatch
	job.Tasks = []*models.Task{
		{
			Name: "task1",
			Engine: &models.SpecConfig{
				Type: models.EngineDocker,
			},
		},
	}

	// Create the job
	err := s.store.CreateJob(s.ctx, *job)
	s.Require().NoError(err)

	// Add job history
	err = s.store.AddJobHistory(s.ctx, job.ID, job.Version, *models.NewEvent("job-created").WithMessage("Job created"))
	s.Require().NoError(err)

	// Test getting job ID with wrong namespace
	jobNameKey := fmt.Sprintf("%s:%s", job.Name, "development") // Wrong namespace

	// Use a transaction to test the private method
	err = boltdblib.View(s.ctx, s.store.database, func(tx *bolt.Tx) error {
		recorder := s.store.metricRecorder(s.ctx, "test", "test")

		_, err := s.store.getJobIDByJobName(s.ctx, tx, recorder, jobNameKey)
		s.Require().Error(err)
		s.Require().True(bacerrors.IsErrorWithCode(err, bacerrors.NotFoundError))

		return nil
	})
	s.Require().NoError(err)
}

func (s *BoltJobstoreTestSuite) TestGetJobIDByJobNameNamespaceIsolation() {
	// Create two jobs with same name in different namespaces
	job1 := mock.Job()
	job1.ID = "test-job-isolation-1"
	job1.Name = "isolated-job"
	job1.Namespace = "development"
	job1.Type = models.JobTypeBatch
	job1.Tasks = []*models.Task{
		{
			Name: "task1",
			Engine: &models.SpecConfig{
				Type: models.EngineDocker,
			},
		},
	}

	job2 := mock.Job()
	job2.ID = "test-job-isolation-2"
	job2.Name = "isolated-job" // Same name as job1
	job2.Namespace = "production"
	job2.Type = models.JobTypeBatch
	job2.Tasks = []*models.Task{
		{
			Name: "task1",
			Engine: &models.SpecConfig{
				Type: models.EngineDocker,
			},
		},
	}

	// Create both jobs
	err := s.store.CreateJob(s.ctx, *job1)
	s.Require().NoError(err)
	err = s.store.CreateJob(s.ctx, *job2)
	s.Require().NoError(err)

	// Add job history for both
	err = s.store.AddJobHistory(s.ctx, job1.ID, job1.Version, *models.NewEvent("job-created").WithMessage("Job 1 created"))
	s.Require().NoError(err)
	err = s.store.AddJobHistory(s.ctx, job2.ID, job2.Version, *models.NewEvent("job-created").WithMessage("Job 2 created"))
	s.Require().NoError(err)

	// Test getting job ID for job1 in development namespace
	jobNameKey1 := fmt.Sprintf("%s:%s", job1.Name, "development")

	err = boltdblib.View(s.ctx, s.store.database, func(tx *bolt.Tx) error {
		recorder := s.store.metricRecorder(s.ctx, "test", "test")

		retrievedJobID, err := s.store.getJobIDByJobName(s.ctx, tx, recorder, jobNameKey1)
		s.Require().NoError(err)
		s.Require().Equal(job1.ID, retrievedJobID)

		return nil
	})
	s.Require().NoError(err)

	// Test getting job ID for job2 in production namespace
	jobNameKey2 := fmt.Sprintf("%s:%s", job2.Name, "production")

	err = boltdblib.View(s.ctx, s.store.database, func(tx *bolt.Tx) error {
		recorder := s.store.metricRecorder(s.ctx, "test", "test")

		retrievedJobID, err := s.store.getJobIDByJobName(s.ctx, tx, recorder, jobNameKey2)
		s.Require().NoError(err)
		s.Require().Equal(job2.ID, retrievedJobID)

		return nil
	})
	s.Require().NoError(err)
}

func (s *BoltJobstoreTestSuite) TestGetJobIDByJobNameCaseSensitive() {
	// Create a job
	job := mock.Job()
	job.ID = "test-job-case-sensitive"
	job.Name = "CaseSensitiveJob"
	job.Namespace = "test"
	job.Type = models.JobTypeBatch
	job.Tasks = []*models.Task{
		{
			Name: "task1",
			Engine: &models.SpecConfig{
				Type: models.EngineDocker,
			},
		},
	}

	// Create the job
	err := s.store.CreateJob(s.ctx, *job)
	s.Require().NoError(err)

	// Add job history
	err = s.store.AddJobHistory(s.ctx, job.ID, job.Version, *models.NewEvent("job-created").WithMessage("Job created"))
	s.Require().NoError(err)

	// Test getting job ID with exact case
	jobNameKeyExact := fmt.Sprintf("%s:%s", "CaseSensitiveJob", job.Namespace)

	err = boltdblib.View(s.ctx, s.store.database, func(tx *bolt.Tx) error {
		recorder := s.store.metricRecorder(s.ctx, "test", "test")

		retrievedJobID, err := s.store.getJobIDByJobName(s.ctx, tx, recorder, jobNameKeyExact)
		s.Require().NoError(err)
		s.Require().Equal(job.ID, retrievedJobID)

		return nil
	})
	s.Require().NoError(err)

	// Test getting job ID with different case (should not be found)
	jobNameKeyWrongCase := fmt.Sprintf("%s:%s", "casesensitivejob", job.Namespace)

	err = boltdblib.View(s.ctx, s.store.database, func(tx *bolt.Tx) error {
		recorder := s.store.metricRecorder(s.ctx, "test", "test")

		_, err := s.store.getJobIDByJobName(s.ctx, tx, recorder, jobNameKeyWrongCase)
		s.Require().Error(err)
		s.Require().True(bacerrors.IsErrorWithCode(err, bacerrors.NotFoundError))

		return nil
	})
	s.Require().NoError(err)
}

func (s *BoltJobstoreTestSuite) TestGetJobIDByJobNameSpecialCharacters() {
	// Create a job with special characters in the name
	job := mock.Job()
	job.ID = "test-job-special-chars"
	job.Name = "job-with-special_chars.and@symbols"
	job.Namespace = "test"
	job.Type = models.JobTypeBatch
	job.Tasks = []*models.Task{
		{
			Name: "task1",
			Engine: &models.SpecConfig{
				Type: models.EngineDocker,
			},
		},
	}

	// Create the job
	err := s.store.CreateJob(s.ctx, *job)
	s.Require().NoError(err)

	// Add job history
	err = s.store.AddJobHistory(s.ctx, job.ID, job.Version, *models.NewEvent("job-created").WithMessage("Job created"))
	s.Require().NoError(err)

	// Test getting job ID with special characters
	jobNameKey := fmt.Sprintf("%s:%s", job.Name, job.Namespace)

	err = boltdblib.View(s.ctx, s.store.database, func(tx *bolt.Tx) error {
		recorder := s.store.metricRecorder(s.ctx, "test", "test")

		retrievedJobID, err := s.store.getJobIDByJobName(s.ctx, tx, recorder, jobNameKey)
		s.Require().NoError(err)
		s.Require().Equal(job.ID, retrievedJobID)

		return nil
	})
	s.Require().NoError(err)
}

func (s *BoltJobstoreTestSuite) TestGetJobIDByJobNameInvalidKey() {
	// Test with malformed job name key (missing colon separator)
	invalidJobNameKey := "invalid-key-without-colon"

	err := boltdblib.View(s.ctx, s.store.database, func(tx *bolt.Tx) error {
		recorder := s.store.metricRecorder(s.ctx, "test", "test")

		_, err := s.store.getJobIDByJobName(s.ctx, tx, recorder, invalidJobNameKey)
		s.Require().Error(err)
		s.Require().True(bacerrors.IsErrorWithCode(err, bacerrors.NotFoundError))

		return nil
	})
	s.Require().NoError(err)
}

func (s *BoltJobstoreTestSuite) TestGetJobIDByJobNameEmptyKey() {
	// Test with empty job name key
	emptyJobNameKey := ""

	err := boltdblib.View(s.ctx, s.store.database, func(tx *bolt.Tx) error {
		recorder := s.store.metricRecorder(s.ctx, "test", "test")

		_, err := s.store.getJobIDByJobName(s.ctx, tx, recorder, emptyJobNameKey)
		s.Require().Error(err)
		s.Require().True(bacerrors.IsErrorWithCode(err, bacerrors.NotFoundError))

		return nil
	})
	s.Require().NoError(err)
}

func (s *BoltJobstoreTestSuite) TestJobExistsByName() {
	// Create a job
	job := mock.Job()
	job.ID = "test-job-exists-by-name-basic"
	job.Name = "basic-exists-job"
	job.Namespace = "test"
	job.Type = models.JobTypeBatch
	job.Tasks = []*models.Task{
		{
			Name: "task1",
			Engine: &models.SpecConfig{
				Type: models.EngineDocker,
			},
		},
	}

	// Create the job
	err := s.store.CreateJob(s.ctx, *job)
	s.Require().NoError(err)

	// Add job history
	err = s.store.AddJobHistory(s.ctx, job.ID, job.Version, *models.NewEvent("job-created").WithMessage("Job created"))
	s.Require().NoError(err)

	// Test that job exists by name
	err = boltdblib.View(s.ctx, s.store.database, func(tx *bolt.Tx) error {
		recorder := s.store.metricRecorder(s.ctx, "test", "test")

		exists := s.store.jobExistsByName(s.ctx, tx, recorder, job.Name, job.Namespace)
		s.Require().True(exists)

		return nil
	})
	s.Require().NoError(err)
}

func (s *BoltJobstoreTestSuite) TestJobExistsByNameWithDefaultNamespace() {
	// Create a job with default namespace
	job := mock.Job()
	job.ID = "test-job-exists-default-ns"
	job.Name = "default-namespace-exists-job"
	job.Namespace = models.DefaultNamespace
	job.Type = models.JobTypeBatch
	job.Tasks = []*models.Task{
		{
			Name: "task1",
			Engine: &models.SpecConfig{
				Type: models.EngineDocker,
			},
		},
	}

	// Create the job
	err := s.store.CreateJob(s.ctx, *job)
	s.Require().NoError(err)

	// Add job history
	err = s.store.AddJobHistory(s.ctx, job.ID, job.Version, *models.NewEvent("job-created").WithMessage("Job created"))
	s.Require().NoError(err)

	// Test that job exists by name with default namespace
	err = boltdblib.View(s.ctx, s.store.database, func(tx *bolt.Tx) error {
		recorder := s.store.metricRecorder(s.ctx, "test", "test")

		exists := s.store.jobExistsByName(s.ctx, tx, recorder, job.Name, models.DefaultNamespace)
		s.Require().True(exists)

		return nil
	})
	s.Require().NoError(err)

	// Test that job exists by name with empty namespace (should default to DefaultNamespace)
	err = boltdblib.View(s.ctx, s.store.database, func(tx *bolt.Tx) error {
		recorder := s.store.metricRecorder(s.ctx, "test", "test")

		exists := s.store.jobExistsByName(s.ctx, tx, recorder, job.Name, "")
		s.Require().True(exists)

		return nil
	})
	s.Require().NoError(err)
}

func (s *BoltJobstoreTestSuite) TestJobExistsByNameWithCustomNamespace() {
	// Create a job with custom namespace
	job := mock.Job()
	job.ID = "test-job-exists-custom-ns"
	job.Name = "custom-namespace-exists-job"
	job.Namespace = "production"
	job.Type = models.JobTypeBatch
	job.Tasks = []*models.Task{
		{
			Name: "task1",
			Engine: &models.SpecConfig{
				Type: models.EngineDocker,
			},
		},
	}

	// Create the job
	err := s.store.CreateJob(s.ctx, *job)
	s.Require().NoError(err)

	// Add job history
	err = s.store.AddJobHistory(s.ctx, job.ID, job.Version, *models.NewEvent("job-created").WithMessage("Job created"))
	s.Require().NoError(err)

	// Test that job exists by name with custom namespace
	err = boltdblib.View(s.ctx, s.store.database, func(tx *bolt.Tx) error {
		recorder := s.store.metricRecorder(s.ctx, "test", "test")

		exists := s.store.jobExistsByName(s.ctx, tx, recorder, job.Name, "production")
		s.Require().True(exists)

		return nil
	})
	s.Require().NoError(err)
}

func (s *BoltJobstoreTestSuite) TestJobExistsByNameNotFound() {
	// Test that non-existent job returns false
	err := boltdblib.View(s.ctx, s.store.database, func(tx *bolt.Tx) error {
		recorder := s.store.metricRecorder(s.ctx, "test", "test")

		exists := s.store.jobExistsByName(s.ctx, tx, recorder, "non-existent-job", "test")
		s.Require().False(exists)

		return nil
	})
	s.Require().NoError(err)
}

func (s *BoltJobstoreTestSuite) TestJobExistsByNameWrongNamespace() {
	// Create a job in one namespace
	job := mock.Job()
	job.ID = "test-job-exists-wrong-ns"
	job.Name = "namespace-exists-job"
	job.Namespace = "production"
	job.Type = models.JobTypeBatch
	job.Tasks = []*models.Task{
		{
			Name: "task1",
			Engine: &models.SpecConfig{
				Type: models.EngineDocker,
			},
		},
	}

	// Create the job
	err := s.store.CreateJob(s.ctx, *job)
	s.Require().NoError(err)

	// Add job history
	err = s.store.AddJobHistory(s.ctx, job.ID, job.Version, *models.NewEvent("job-created").WithMessage("Job created"))
	s.Require().NoError(err)

	// Test that job exists in correct namespace
	err = boltdblib.View(s.ctx, s.store.database, func(tx *bolt.Tx) error {
		recorder := s.store.metricRecorder(s.ctx, "test", "test")

		exists := s.store.jobExistsByName(s.ctx, tx, recorder, job.Name, "production")
		s.Require().True(exists)

		return nil
	})
	s.Require().NoError(err)

	// Test that job does not exist in wrong namespace
	err = boltdblib.View(s.ctx, s.store.database, func(tx *bolt.Tx) error {
		recorder := s.store.metricRecorder(s.ctx, "test", "test")

		exists := s.store.jobExistsByName(s.ctx, tx, recorder, job.Name, "development")
		s.Require().False(exists)

		return nil
	})
	s.Require().NoError(err)
}

func (s *BoltJobstoreTestSuite) TestJobExistsByNameNamespaceIsolation() {
	// Create two jobs with same name in different namespaces
	job1 := mock.Job()
	job1.ID = "test-job-exists-isolation-1"
	job1.Name = "isolated-exists-job"
	job1.Namespace = "development"
	job1.Type = models.JobTypeBatch
	job1.Tasks = []*models.Task{
		{
			Name: "task1",
			Engine: &models.SpecConfig{
				Type: models.EngineDocker,
			},
		},
	}

	job2 := mock.Job()
	job2.ID = "test-job-exists-isolation-2"
	job2.Name = "isolated-exists-job" // Same name as job1
	job2.Namespace = "production"
	job2.Type = models.JobTypeBatch
	job2.Tasks = []*models.Task{
		{
			Name: "task1",
			Engine: &models.SpecConfig{
				Type: models.EngineDocker,
			},
		},
	}

	// Create both jobs
	err := s.store.CreateJob(s.ctx, *job1)
	s.Require().NoError(err)
	err = s.store.CreateJob(s.ctx, *job2)
	s.Require().NoError(err)

	// Add job history for both
	err = s.store.AddJobHistory(s.ctx, job1.ID, job1.Version, *models.NewEvent("job-created").WithMessage("Job 1 created"))
	s.Require().NoError(err)
	err = s.store.AddJobHistory(s.ctx, job2.ID, job2.Version, *models.NewEvent("job-created").WithMessage("Job 2 created"))
	s.Require().NoError(err)

	// Test that job1 exists in development namespace
	err = boltdblib.View(s.ctx, s.store.database, func(tx *bolt.Tx) error {
		recorder := s.store.metricRecorder(s.ctx, "test", "test")

		exists := s.store.jobExistsByName(s.ctx, tx, recorder, job1.Name, "development")
		s.Require().True(exists)

		return nil
	})
	s.Require().NoError(err)

	// Test that job2 exists in production namespace
	err = boltdblib.View(s.ctx, s.store.database, func(tx *bolt.Tx) error {
		recorder := s.store.metricRecorder(s.ctx, "test", "test")

		exists := s.store.jobExistsByName(s.ctx, tx, recorder, job2.Name, "production")
		s.Require().True(exists)

		return nil
	})
	s.Require().NoError(err)

	// Test that job with same name exists in production namespace (job2 exists there)
	err = boltdblib.View(s.ctx, s.store.database, func(tx *bolt.Tx) error {
		recorder := s.store.metricRecorder(s.ctx, "test", "test")

		exists := s.store.jobExistsByName(s.ctx, tx, recorder, job1.Name, "production")
		s.Require().True(exists) // This should be true because job2 has the same name

		return nil
	})
	s.Require().NoError(err)

	// Test that job with same name exists in development namespace (job1 exists there)
	err = boltdblib.View(s.ctx, s.store.database, func(tx *bolt.Tx) error {
		recorder := s.store.metricRecorder(s.ctx, "test", "test")

		exists := s.store.jobExistsByName(s.ctx, tx, recorder, job2.Name, "development")
		s.Require().True(exists) // This should be true because job1 has the same name

		return nil
	})
	s.Require().NoError(err)
}

func (s *BoltJobstoreTestSuite) TestJobExistsByNameCaseSensitive() {
	// Create a job
	job := mock.Job()
	job.ID = "test-job-exists-case-sensitive"
	job.Name = "CaseSensitiveExistsJob"
	job.Namespace = "test"
	job.Type = models.JobTypeBatch
	job.Tasks = []*models.Task{
		{
			Name: "task1",
			Engine: &models.SpecConfig{
				Type: models.EngineDocker,
			},
		},
	}

	// Create the job
	err := s.store.CreateJob(s.ctx, *job)
	s.Require().NoError(err)

	// Add job history
	err = s.store.AddJobHistory(s.ctx, job.ID, job.Version, *models.NewEvent("job-created").WithMessage("Job created"))
	s.Require().NoError(err)

	// Test that job exists with exact case
	err = boltdblib.View(s.ctx, s.store.database, func(tx *bolt.Tx) error {
		recorder := s.store.metricRecorder(s.ctx, "test", "test")

		exists := s.store.jobExistsByName(s.ctx, tx, recorder, "CaseSensitiveExistsJob", job.Namespace)
		s.Require().True(exists)

		return nil
	})
	s.Require().NoError(err)

	// Test that job does not exist with different case
	err = boltdblib.View(s.ctx, s.store.database, func(tx *bolt.Tx) error {
		recorder := s.store.metricRecorder(s.ctx, "test", "test")

		exists := s.store.jobExistsByName(s.ctx, tx, recorder, "casesensitiveexistsjob", job.Namespace)
		s.Require().False(exists)

		return nil
	})
	s.Require().NoError(err)
}

func (s *BoltJobstoreTestSuite) TestJobExistsByNameSpecialCharacters() {
	// Create a job with special characters in the name
	job := mock.Job()
	job.ID = "test-job-exists-special-chars"
	job.Name = "job-with-special_chars.and@symbols-exists"
	job.Namespace = "test"
	job.Type = models.JobTypeBatch
	job.Tasks = []*models.Task{
		{
			Name: "task1",
			Engine: &models.SpecConfig{
				Type: models.EngineDocker,
			},
		},
	}

	// Create the job
	err := s.store.CreateJob(s.ctx, *job)
	s.Require().NoError(err)

	// Add job history
	err = s.store.AddJobHistory(s.ctx, job.ID, job.Version, *models.NewEvent("job-created").WithMessage("Job created"))
	s.Require().NoError(err)

	// Test that job exists with special characters
	err = boltdblib.View(s.ctx, s.store.database, func(tx *bolt.Tx) error {
		recorder := s.store.metricRecorder(s.ctx, "test", "test")

		exists := s.store.jobExistsByName(s.ctx, tx, recorder, job.Name, job.Namespace)
		s.Require().True(exists)

		return nil
	})
	s.Require().NoError(err)
}

func (s *BoltJobstoreTestSuite) TestJobExistsByNameEmptyName() {
	// Test with empty job name
	err := boltdblib.View(s.ctx, s.store.database, func(tx *bolt.Tx) error {
		recorder := s.store.metricRecorder(s.ctx, "test", "test")

		exists := s.store.jobExistsByName(s.ctx, tx, recorder, "", "test")
		s.Require().False(exists)

		return nil
	})
	s.Require().NoError(err)
}

func (s *BoltJobstoreTestSuite) TestJobExistsByNameMultipleJobs() {
	// Create multiple jobs with different names in the same namespace
	job1 := mock.Job()
	job1.ID = "test-job-exists-multiple-1"
	job1.Name = "multiple-exists-job-1"
	job1.Namespace = "test"
	job1.Type = models.JobTypeBatch
	job1.Tasks = []*models.Task{
		{
			Name: "task1",
			Engine: &models.SpecConfig{
				Type: models.EngineDocker,
			},
		},
	}

	job2 := mock.Job()
	job2.ID = "test-job-exists-multiple-2"
	job2.Name = "multiple-exists-job-2"
	job2.Namespace = "test"
	job2.Type = models.JobTypeBatch
	job2.Tasks = []*models.Task{
		{
			Name: "task1",
			Engine: &models.SpecConfig{
				Type: models.EngineDocker,
			},
		},
	}

	job3 := mock.Job()
	job3.ID = "test-job-exists-multiple-3"
	job3.Name = "multiple-exists-job-3"
	job3.Namespace = "test"
	job3.Type = models.JobTypeBatch
	job3.Tasks = []*models.Task{
		{
			Name: "task1",
			Engine: &models.SpecConfig{
				Type: models.EngineDocker,
			},
		},
	}

	// Create all jobs
	err := s.store.CreateJob(s.ctx, *job1)
	s.Require().NoError(err)
	err = s.store.CreateJob(s.ctx, *job2)
	s.Require().NoError(err)
	err = s.store.CreateJob(s.ctx, *job3)
	s.Require().NoError(err)

	// Add job history for all
	err = s.store.AddJobHistory(s.ctx, job1.ID, job1.Version, *models.NewEvent("job-created").WithMessage("Job 1 created"))
	s.Require().NoError(err)
	err = s.store.AddJobHistory(s.ctx, job2.ID, job2.Version, *models.NewEvent("job-created").WithMessage("Job 2 created"))
	s.Require().NoError(err)
	err = s.store.AddJobHistory(s.ctx, job3.ID, job3.Version, *models.NewEvent("job-created").WithMessage("Job 3 created"))
	s.Require().NoError(err)

	// Test that all jobs exist
	err = boltdblib.View(s.ctx, s.store.database, func(tx *bolt.Tx) error {
		recorder := s.store.metricRecorder(s.ctx, "test", "test")

		exists1 := s.store.jobExistsByName(s.ctx, tx, recorder, job1.Name, "test")
		s.Require().True(exists1)

		exists2 := s.store.jobExistsByName(s.ctx, tx, recorder, job2.Name, "test")
		s.Require().True(exists2)

		exists3 := s.store.jobExistsByName(s.ctx, tx, recorder, job3.Name, "test")
		s.Require().True(exists3)

		// Test that non-existent job does not exist
		existsNon := s.store.jobExistsByName(s.ctx, tx, recorder, "non-existent-job", "test")
		s.Require().False(existsNon)

		return nil
	})
	s.Require().NoError(err)
}

func (s *BoltJobstoreTestSuite) TestUpdateJobJobNameIndexInconsistency() {
	// Create the first job normally
	job1 := mock.Job()
	job1.ID = "job-id-123"
	job1.Name = "test-job-inconsistency"
	job1.Namespace = "test"
	job1.Type = models.JobTypeBatch
	job1.Tasks = []*models.Task{
		{
			Name: "task1",
			Engine: &models.SpecConfig{
				Type: models.EngineDocker,
			},
		},
	}
	job1.Labels = map[string]string{"version": "1"}

	// Create the job
	err := s.store.CreateJob(s.ctx, *job1)
	s.Require().NoError(err)

	// Create a second job that will be used to corrupt the name index
	job2 := mock.Job()
	job2.ID = "job-id-456"
	job2.Name = "different-job"
	job2.Namespace = "test"
	job2.Type = models.JobTypeBatch
	job2.Tasks = []*models.Task{
		{
			Name: "task2",
			Engine: &models.SpecConfig{
				Type: models.EngineDocker,
			},
		},
	}
	job2.Labels = map[string]string{"version": "1"}

	// Create the second job
	err = s.store.CreateJob(s.ctx, *job2)
	s.Require().NoError(err)

	// Manually corrupt the names index to create the inconsistency
	// This simulates the scenario mentioned in the error message where
	// a job created before version 1.8 might have index inconsistencies
	err = boltdblib.Update(s.ctx, s.store.database, func(tx *bolt.Tx) error {
		// Create a fake entry in the names index that points job1's name
		// to job2's ID, creating the inconsistency
		jobNameKey := createJobNameIndexKey(job1.Name, job1.Namespace)

		// Remove the correct entry for job1
		if err := s.store.namesIndex.Remove(tx, []byte(job1.ID), []byte(jobNameKey)); err != nil {
			return err
		}

		// Add an incorrect entry that maps job1's name to job2's ID
		if err := s.store.namesIndex.Add(tx, []byte(job2.ID), []byte(jobNameKey)); err != nil {
			return err
		}

		return nil
	})
	s.Require().NoError(err)

	// Now try to update job1 - this should trigger the inconsistency error
	updatedJob1 := *job1
	updatedJob1.Labels["version"] = "2"
	updatedJob1.Meta = map[string]string{"updated": "true"}

	err = s.store.UpdateJob(s.ctx, updatedJob1)
	s.Require().Error(err)

	// Verify that the error is of the expected type and contains the expected message
	s.Require().Contains(err.Error(), "inconsistency between the Job name and its ID")
	s.Require().Contains(err.Error(), fmt.Sprintf("Job name %s with ID %s does not match stored job ID %s",
		job1.Name, job1.ID, job2.ID))

	// Verify it's a bacerrors.Error with the hint
	var bacErr bacerrors.Error
	s.Require().True(errors.As(err, &bacErr))
	s.Require().Contains(bacErr.Hint(), "This usually happens if you try to rerun a job, using its ID, that was created before version 1.8")
}
