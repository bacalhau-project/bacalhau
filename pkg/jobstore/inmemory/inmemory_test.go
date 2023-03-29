//go:build unit || !integration

package inmemory

import (
	"context"
	"testing"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type InMemoryTestSuite struct {
	suite.Suite
	store *JobStore
	ctx   context.Context
}

func TestInMemoryTestSuite(t *testing.T) {
	suite.Run(t, new(InMemoryTestSuite))
}

func (s *InMemoryTestSuite) SetupTest() {
	s.store = NewJobStore()
	s.ctx = context.Background()

	var logicalClock int64 = 0

	jobFixtures := []struct {
		id              string
		totalEntries    int
		jobStates       []model.JobStateType
		executionStates []model.ExecutionStateType
	}{
		{
			id:              "1",
			jobStates:       []model.JobStateType{model.JobStateQueued, model.JobStateInProgress, model.JobStateCancelled},
			executionStates: []model.ExecutionStateType{model.ExecutionStateAskForBid, model.ExecutionStateAskForBidAccepted, model.ExecutionStateFailed, model.ExecutionStateCanceled},
		},
	}

	for _, fixture := range jobFixtures {
		for i, state := range fixture.jobStates {
			oldState := model.JobStateNew
			if i > 0 {
				oldState = fixture.jobStates[i-1]
			}

			jobState := model.JobState{
				JobID:      fixture.id,
				State:      state,
				UpdateTime: time.Unix(logicalClock, 0),
			}
			s.store.appendJobHistory(jobState, oldState, "")
			logicalClock += 1
		}

		for i, state := range fixture.executionStates {
			oldState := model.ExecutionStateNew
			if i > 0 {
				oldState = fixture.executionStates[i-1]
			}

			e := model.ExecutionState{

				JobID:      fixture.id,
				State:      state,
				UpdateTime: time.Unix(logicalClock, 0),
			}
			s.store.appendExecutionHistory(e, oldState, "")
			logicalClock += 1
		}
	}

}

func (s *InMemoryTestSuite) TestUnfilteredJobHistory() {
	history, err := s.store.GetJobHistory(s.ctx, "1", jobstore.JobHistoryFilterOptions{})
	require.NoError(s.T(), err, "failed to get job history")
	require.Equal(s.T(), 7, len(history))
}

func (s *InMemoryTestSuite) TestJobHistoryOrdering() {
	history, err := s.store.GetJobHistory(s.ctx, "1", jobstore.JobHistoryFilterOptions{})
	require.NoError(s.T(), err, "failed to get job history")
	require.Equal(s.T(), 7, len(history))

	values := make([]int64, len(history))
	for i, h := range history {
		values[i] = h.Time.Unix()
	}

	require.Equal(s.T(), []int64{0, 1, 2, 3, 4, 5, 6}, values)
}

func (s *InMemoryTestSuite) TestTimeFilteredJobHistory() {
	options := jobstore.JobHistoryFilterOptions{
		Since: 3,
	}

	history, err := s.store.GetJobHistory(s.ctx, "1", options)
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

	history, err := s.store.GetJobHistory(s.ctx, "1", jobOptions)
	require.NoError(s.T(), err, "failed to get job history")
	require.Equal(s.T(), 3, len(history))
	require.Equal(s.T(), model.JobStateQueued, history[0].JobState.New)

	history, err = s.store.GetJobHistory(s.ctx, "1", execOptions)
	require.NoError(s.T(), err, "failed to get job history")
	require.Equal(s.T(), 4, len(history))
	require.Equal(s.T(), model.ExecutionStateAskForBid, history[0].ExecutionState.New)
}
