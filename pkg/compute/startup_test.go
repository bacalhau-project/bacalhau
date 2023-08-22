//go:build unit || !integration

package compute_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"
	"go.uber.org/mock/gomock"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	"github.com/bacalhau-project/bacalhau/pkg/compute/store/inmemory"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/test/mock"
	testutils "github.com/bacalhau-project/bacalhau/pkg/test/utils"
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
		ID           string
		job          model.Job
		allNodes     bool
		expectations func(_ *compute.MockExecutor)
	}

	any := gomock.Any()

	testcases := []testcase{
		{
			ID:       "1",
			job:      *testutils.MakeNoopJob(s.T()),
			allNodes: false,
			expectations: func(executor *compute.MockExecutor) {
				executor.EXPECT().Cancel(any, any).Return(nil)
			},
		},
		{
			ID:       "2",
			job:      *testutils.MakeNoopJob(s.T()),
			allNodes: true,
			expectations: func(executor *compute.MockExecutor) {
				// Need to specify max times to include the call in the next text case,
				// but isn't clear why given the new controller and mock created in
				// each test case.
				executor.EXPECT().Cancel(any, any).Return(nil).MaxTimes(2)
				executor.EXPECT().Run(any, any).Return(nil)
			},
		},
	}

	for idx, tc := range testcases {
		s.T().Run(fmt.Sprintf("Test %d", idx+1), func(t *testing.T) {
			ctrl := gomock.NewController(t)
			mockExecutor := compute.NewMockExecutor(ctrl)

			tc.expectations(mockExecutor) // check expectations

			j := tc.job
			if tc.allNodes {
				j.Spec.Deal.TargetingMode = model.TargetAll
			}

			execution := mock.ExecutionForJob(mock.Job())
			execution.ID = tc.ID
			exec := store.NewLocalExecutionState(execution, "req")
			err := database.CreateExecution(s.ctx, *exec)
			s.Require().NoError(err)

			err = database.UpdateExecutionState(s.ctx, store.UpdateExecutionStateRequest{
				ExecutionID: execution.ID,
				NewState:    store.ExecutionStateRunning,
			})
			s.Require().NoError(err)

			execs, err := database.GetLiveExecutions(s.ctx)
			s.Require().NoError(err)
			s.Require().Equal(idx+1, len(execs))

			startup := compute.NewStartup(database, mockExecutor)
			err = startup.Execute(s.ctx)
			s.Require().NoError(err)

			// If we get here we're good as mock expectations didn't fail.
			ctrl.Finish()
		})
	}
}
