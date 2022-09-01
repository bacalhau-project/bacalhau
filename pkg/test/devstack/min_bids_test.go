package devstack

import (
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/computenode"
	"github.com/filecoin-project/bacalhau/pkg/job"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/publicapi"
	"github.com/filecoin-project/bacalhau/pkg/system"
	testutils "github.com/filecoin-project/bacalhau/pkg/test/utils"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type MinBidsSuite struct {
	suite.Suite
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestMinBidsSuite(t *testing.T) {
	suite.Run(t, new(MinBidsSuite))
}

// Before all suite
func (suite *MinBidsSuite) SetupAllSuite() {

}

// Before each test
func (suite *MinBidsSuite) SetupTest() {
	system.InitConfigForTesting(suite.T())
}

func (suite *MinBidsSuite) TearDownTest() {
}

func (suite *MinBidsSuite) TearDownAllSuite() {

}

type minBidsTestCase struct {
	nodes          int
	shards         int
	concurrency    int
	minBids        int
	expectedResult map[model.JobStateType]int
}

func (suite *MinBidsSuite) TestMinBids() {

	runTest := func(
		testCase minBidsTestCase,
	) {
		ctx, span := newSpan("TestMinBids")
		defer span.End()

		stack, cm := SetupTest(
			suite.T(),
			testCase.nodes,
			0,
			computenode.NewDefaultComputeNodeConfig(),
		)
		defer TeardownTest(stack, cm)

		dirPath, err := prepareFolderWithFiles(testCase.shards)
		require.NoError(suite.T(), err)

		directoryCid, err := stack.AddFileToNodes(testCase.nodes, dirPath)
		require.NoError(suite.T(), err)

		apiUri := stack.Nodes[0].APIServer.GetURI()
		apiClient := publicapi.NewAPIClient(apiUri)

		spec := testutils.DockerRunJob()
		spec.Inputs = []model.StorageSpec{
			{
				Engine: model.StorageSourceIPFS,
				Cid:    directoryCid,
				Path:   "/input",
			},
		}
		spec.Sharding = model.JobShardingConfig{
			GlobPattern: "/input/*",
			BatchSize:   1,
		}

		deal := model.JobDeal{
			Concurrency: testCase.concurrency,
			MinBids:     testCase.minBids,
		}

		createdJob, err := apiClient.Submit(ctx, spec, deal, nil)
		require.NoError(suite.T(), err)
		resolver := apiClient.GetJobStateResolver()

		err = resolver.Wait(
			ctx,
			createdJob.ID,
			3,
			job.WaitThrowErrors([]model.JobStateType{
				model.JobStateError,
			}),
			job.WaitForJobStates(testCase.expectedResult),
		)
		require.NoError(suite.T(), err)

	}

	// sanity test that with min bids at zero and 1 node we get the job through
	runTest(minBidsTestCase{
		nodes:       1,
		shards:      1,
		concurrency: 1,
		minBids:     0,
		expectedResult: map[model.JobStateType]int{
			model.JobStatePublished: 1,
		},
	})

}
