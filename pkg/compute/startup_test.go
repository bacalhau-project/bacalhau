//go:build unit || !integration

package compute_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	"github.com/bacalhau-project/bacalhau/pkg/compute/store/inmemory"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/test/mock"
)

type StartupTestSuite struct {
	suite.Suite
	ctx context.Context
}

func TestStartupTestSuite(t *testing.T) {
	suite.Run(t, new(StartupTestSuite))
}

func (s *StartupTestSuite) SetupTest() {
	s.ctx = context.Background()
}

func (s *StartupTestSuite) TestLongRunning() {

	database := inmemory.NewStore()
	defer database.Close(s.ctx)

	type testcase struct {
		ID       string
		job_type string
		allNodes bool
	}

	any := gomock.Any()

	testcases := []testcase{
		{
			ID:       "1",
			job_type: models.JobTypeBatch,
			allNodes: false,
		},
		{
			ID:       "2",
			job_type: models.JobTypeService,
			allNodes: true,
		},
	}

	// Create jobs and live executions for test cases.
	for _, tc := range testcases {
		j := mock.Job()
		j.ID = tc.ID
		j.Type = tc.job_type

		execution := mock.ExecutionForJob(j)
		execution.ID = tc.ID
		exec := store.NewLocalExecutionState(execution, "req")
		err := database.CreateExecution(s.ctx, *exec)
		s.Require().NoError(err)

		err = database.UpdateExecutionState(s.ctx, store.UpdateExecutionStateRequest{
			ExecutionID: execution.ID,
			NewState:    store.ExecutionStateRunning,
		})
		s.Require().NoError(err)

	}

	ctrl := gomock.NewController(s.T())
	mockExecutor := compute.NewMockExecutor(ctrl)

	mockExecutor.EXPECT().Cancel(any, any).Return(nil).MaxTimes(2)
	mockExecutor.EXPECT().Run(any, any).Return(nil)

	execs, err := database.GetLiveExecutions(s.ctx)
	s.Require().NoError(err)
	s.Require().Equal(2, len(execs))

	startup := compute.NewStartup(database, mockExecutor)
	err = startup.Execute(s.ctx)
	s.Require().NoError(err)

	// If we get here we're good as mock expectations didn't fail.
	ctrl.Finish()
}
