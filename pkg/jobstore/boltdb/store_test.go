//go:build unit || !integration

package boltjobstore

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	testutils "github.com/bacalhau-project/bacalhau/pkg/test/utils"
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
		totalEntries    int
		jobStates       []model.JobStateType
		executionStates []model.ExecutionStateType
	}{
		{
			id:              "1",
			jobStates:       []model.JobStateType{model.JobStateQueued, model.JobStateInProgress, model.JobStateCancelled},
			executionStates: []model.ExecutionStateType{model.ExecutionStateAskForBid, model.ExecutionStateAskForBidAccepted, model.ExecutionStateCancelled},
		},
	}

	for _, fixture := range jobFixtures {
		s.clock.Add(1 * time.Second)
		job := testutils.MakeJob(
			model.EngineDocker,
			model.PublisherNoop,
			[]string{"bash", "-c", "echo hello"})
		job.Metadata.ID = fixture.id
		job.Metadata.ClientID = "client"
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
					ExpectedState:   oldState,
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

func (s *BoltJobstoreTestSuite) TestDeleteJob() {
	job := testutils.MakeJob(
		model.EngineDocker,
		model.PublisherNoop,
		[]string{"bash", "-c", "echo hello"})
	job.Metadata.ID = "deleteme"
	job.Metadata.ClientID = "client"
	err := s.store.CreateJob(s.ctx, *job)
	s.NoError(err)

	err = s.store.DeleteJob(s.ctx, job.Metadata.ID)
	s.NoError(err)
}
