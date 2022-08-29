package devstack

import (
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/computenode"
	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/job"
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

	createdJob, err := apiClient.Submit(ctx, jobSpec, jobDeal, nil)
	require.NoError(suite.T(), err)

	resolver := apiClient.GetJobStateResolver()

	err = resolver.Wait(
		ctx,
		createdJob.ID,
		3,
		job.WaitThrowErrors([]executor.JobStateType{
			executor.JobStateError,
		}),
		job.WaitForJobStates(map[executor.JobStateType]int{
			executor.JobStateComplete:  2,
			executor.JobStateCancelled: 1,
		}),
	)
	require.NoError(suite.T(), err)
}
