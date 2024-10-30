//go:build unit || !integration

package compute_test

import (
	"context"
	"errors"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	"github.com/bacalhau-project/bacalhau/pkg/compute/store/boltdb"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/test/mock"
)

type StartupTestSuite struct {
	suite.Suite
	ctx          context.Context
	database     store.ExecutionStore
	mockExecutor *compute.MockExecutor
	startup      *compute.Startup
}

func TestStartupTestSuite(t *testing.T) {
	suite.Run(t, new(StartupTestSuite))
}

func (s *StartupTestSuite) SetupTest() {
	s.ctx = context.Background()
	var err error
	s.database, err = boltdb.NewStore(s.ctx, filepath.Join(s.T().TempDir(), "startup-test.db"))
	s.Require().NoError(err)

	ctrl := gomock.NewController(s.T())
	s.mockExecutor = compute.NewMockExecutor(ctrl)

	s.startup = compute.NewStartup(s.database, s.mockExecutor)
}

func (s *StartupTestSuite) TearDownTest() {
	s.database.Close(s.ctx)
}

func (s *StartupTestSuite) TestEnsureLiveJobs() {
	testCases := []struct {
		name      string
		jobType   string
		expectRun bool
	}{
		{
			name:      "BatchJob",
			jobType:   models.JobTypeBatch,
			expectRun: false,
		},
		{
			name:      "ServiceJob",
			jobType:   models.JobTypeService,
			expectRun: true,
		},
		{
			name:      "DaemonJob",
			jobType:   models.JobTypeDaemon,
			expectRun: true,
		},
		{
			name:      "OpsJob",
			jobType:   models.JobTypeOps,
			expectRun: false,
		},
	}

	for _, tc := range testCases {
		s.Run(tc.name, func() {
			s.SetupTest()
			defer s.TearDownTest()

			// Create a job and live execution for the test case
			job := mock.Job()
			job.Type = tc.jobType
			execution := mock.ExecutionForJob(job)

			err := s.database.CreateExecution(s.ctx, *execution)
			s.Require().NoError(err)

			err = s.database.UpdateExecutionState(s.ctx, store.UpdateExecutionRequest{
				ExecutionID: execution.ID,
				NewValues: models.Execution{
					ComputeState: models.NewExecutionState(models.ExecutionStateRunning),
				},
			})
			s.Require().NoError(err)

			if tc.expectRun {
				s.mockExecutor.EXPECT().Run(gomock.Any(), gomock.Any()).Return(nil)
			}

			// Run the startup process
			err = s.startup.Execute(s.ctx)
			s.Require().NoError(err)

			// Verify the execution state after startup
			updatedExec, err := s.database.GetExecution(s.ctx, execution.ID)
			s.Require().NoError(err)

			if tc.expectRun {
				s.Equal(models.ExecutionStateRunning, updatedExec.ComputeState.StateType,
					"expected execution to be %s but was %s", models.ExecutionStateRunning, updatedExec.ComputeState.StateType)
			} else {
				s.Equal(models.ExecutionStateFailed, updatedExec.ComputeState.StateType,
					"expected execution to be %s but was %s", models.ExecutionStateFailed, updatedExec.ComputeState.StateType)
			}
		})
	}
}

func (s *StartupTestSuite) TestEnsureLiveJobsWithError() {
	// Create a service job that will cause an error
	job := mock.Job()
	job.Type = models.JobTypeService
	execution := mock.ExecutionForJob(job)

	err := s.database.CreateExecution(s.ctx, *execution)
	s.Require().NoError(err)

	err = s.database.UpdateExecutionState(s.ctx, store.UpdateExecutionRequest{
		ExecutionID: execution.ID,
		NewValues: models.Execution{
			ComputeState: models.NewExecutionState(models.ExecutionStateRunning),
		},
	})
	s.Require().NoError(err)

	// Expect the Run method to return an error
	s.mockExecutor.EXPECT().Run(gomock.Any(), gomock.Any()).Return(errors.New("execution error"))

	// Run the startup process
	err = s.startup.Execute(s.ctx)
	s.Require().Error(err)

	// Verify that the error didn't prevent other operations
	updatedExec, err := s.database.GetExecution(s.ctx, execution.ID)
	s.Require().NoError(err)
	s.Equal(models.ExecutionStateRunning, updatedExec.ComputeState.StateType)
}
