//go:build unit || !integration

package boltjobstore

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	bolt "go.etcd.io/bbolt"
)

type BoltJobstoreTestSuite struct {
	suite.Suite
	store  *BoltJobStore
	dbFile string
	ctx    context.Context
}

func TestBoltJobstoreTestSuite(t *testing.T) {
	suite.Run(t, new(BoltJobstoreTestSuite))
}

func (s *BoltJobstoreTestSuite) SetupTest() {
	dir, _ := os.MkdirTemp("", "bacalhau-executionstore")
	s.dbFile = filepath.Join(dir, "test.boltdb")

	s.store, _ = NewBoltJobStore(s.dbFile)
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

	// for _, fixture := range jobFixtures {
	// 	job := testutils.MakeJob(
	// 		model.EngineDocker,
	// 		model.VerifierNoop,
	// 		model.PublisherNoop,
	// 		[]string{"bash", "-c", "echo hello"})
	// 	job.Metadata.ID = fixture.id
	// 	s.store.CreateJob(s.ctx, *job)

	// }

	_ = s.store.database.Update(func(tx *bolt.Tx) error {
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

				s.store.appendJobHistory(tx, jobState, oldState, "")
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
				s.store.appendExecutionHistory(tx, e, oldState, "")
				logicalClock += 1
			}
		}
		return nil
	})

}

func (s *BoltJobstoreTestSuite) TearDownTest() {
	s.store.Close(s.ctx)
	os.Remove(s.dbFile)
}

func (s *BoltJobstoreTestSuite) TestUnfilteredJobHistory() {
	history, err := s.store.GetJobHistory(s.ctx, "1", jobstore.JobHistoryFilterOptions{})
	require.NoError(s.T(), err, "failed to get job history")
	require.Equal(s.T(), 7, len(history))
}

func (s *BoltJobstoreTestSuite) TestJobHistoryOrdering() {
	history, err := s.store.GetJobHistory(s.ctx, "1", jobstore.JobHistoryFilterOptions{})
	require.NoError(s.T(), err, "failed to get job history")
	require.Equal(s.T(), 7, len(history))

	values := make([]int64, len(history))
	for i, h := range history {
		values[i] = h.Time.Unix()
	}

	require.Equal(s.T(), []int64{0, 1, 2, 3, 4, 5, 6}, values)
}

func (s *BoltJobstoreTestSuite) TestTimeFilteredJobHistory() {
	options := jobstore.JobHistoryFilterOptions{
		Since: 3,
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
	require.NoError(s.T(), err, "failed to get job history")
	require.Equal(s.T(), 3, len(history))
	require.Equal(s.T(), model.JobStateQueued, history[0].JobState.New)

	history, err = s.store.GetJobHistory(s.ctx, "1", execOptions)
	require.NoError(s.T(), err, "failed to get job history")
	require.Equal(s.T(), 4, len(history))
	require.Equal(s.T(), model.ExecutionStateAskForBid, history[0].ExecutionState.New)
}
