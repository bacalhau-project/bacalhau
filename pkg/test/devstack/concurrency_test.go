package devstack

import (
	"testing"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/computenode"
	"github.com/filecoin-project/bacalhau/pkg/job"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/publicapi"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/test/scenario"
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
	inputStorageList, err := testCase.SetupStorage(stack, model.StorageSourceIPFS, 3)
	require.NoError(suite.T(), err)

	jobSpec := model.JobSpec{
		Engine:    model.EngineDocker,
		Verifier:  model.VerifierNoop,
		Publisher: model.PublisherNoop,
		Docker:    testCase.GetJobSpec(),
		Inputs:    inputStorageList,
		Outputs:   testCase.Outputs,
	}

	jobDeal := model.JobDeal{
		Concurrency: 2,
	}

	apiUri := stack.Nodes[0].APIServer.GetURI()
	apiClient := publicapi.NewAPIClient(apiUri)

	createdJob, err := apiClient.Submit(ctx, jobSpec, jobDeal, nil)
	require.NoError(suite.T(), err)

	resolver := apiClient.GetJobStateResolver()

	stateChecker := func() error {
		return resolver.Wait(
			ctx,
			createdJob.ID,
			3,
			job.WaitThrowErrors([]model.JobStateType{
				model.JobStateError,
			}),
			job.WaitForJobStates(map[model.JobStateType]int{
				model.JobStatePublished: 2,
			}),
		)
	}

	err = stateChecker()
	require.NoError(suite.T(), err)

	// wait a small time and then check again to make sure another JobStatePublished
	// did not sneak in afterwards
	time.Sleep(time.Second * 3)

	err = stateChecker()
	require.NoError(suite.T(), err)
}
