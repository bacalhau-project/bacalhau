package devstack

import (
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/computenode"
	"github.com/filecoin-project/bacalhau/pkg/devstack"
	"github.com/filecoin-project/bacalhau/pkg/executor"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/publicapi"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/test/scenario"
	"github.com/filecoin-project/bacalhau/pkg/verifier"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type DevstackConcurrencySuite struct {
	suite.Suite
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestDevstackConcurrencySuite(t *testing.T) {
	suite.Run(t, new(DevstackConcurrencySuite))
}

// Before all suite
func (suite *DevstackConcurrencySuite) SetupAllSuite() {

}

// Before each test
func (suite *DevstackConcurrencySuite) SetupTest() {
	system.InitConfigForTesting(suite.T())
}

func (suite *DevstackConcurrencySuite) TearDownTest() {
}

func (suite *DevstackConcurrencySuite) TearDownAllSuite() {

}

func (suite *DevstackConcurrencySuite) TestConcurrencyLimit() {
	ctx, span := newSpan("TestConcurrencyLimit")
	defer span.End()

	stack, cm := SetupTest(
		suite.T(),
		3,
		0,
		computenode.NewDefaultComputeNodeConfig(),
	)
	defer TeardownTest(stack, cm)

	testCase := scenario.CatFileToVolume(suite.T())
	inputStorageList, err := testCase.SetupStorage(stack, storage.StorageSourceIPFS, 3)
	require.NoError(suite.T(), err)

	jobSpec := executor.JobSpec{
		Engine:   executor.EngineDocker,
		Verifier: verifier.VerifierIpfs,
		Docker:   testCase.GetJobSpec(),
		Inputs:   inputStorageList,
		Outputs:  testCase.Outputs,
	}

	jobDeal := executor.JobDeal{
		Concurrency: 2,
	}

	apiUri := stack.Nodes[0].APIServer.GetURI()
	apiClient := publicapi.NewAPIClient(apiUri)

	job, err := apiClient.Submit(ctx, jobSpec, jobDeal, nil)
	require.NoError(suite.T(), err)

	err = stack.WaitForJob(ctx, job.ID,
		devstack.WaitForJobThrowErrors([]executor.JobStateType{
			executor.JobStateError,
		}),
		func(jobStates map[string]executor.JobStateType) (bool, error) {
			// we should have 3 states - 2 of which are complete and the other is cancelled
			if len(jobStates) != 3 {
				return false, nil
			}
			completeCount := 0
			cancelledCount := 0
			for _, state := range jobStates {
				if state == executor.JobStateComplete {
					completeCount++
				} else if state == executor.JobStateCancelled {
					cancelledCount++
				}
			}
			if completeCount == 2 && cancelledCount == 1 {
				return true, nil
			} else {
				return false, nil
			}
		},
	)
	require.NoError(suite.T(), err)
}
