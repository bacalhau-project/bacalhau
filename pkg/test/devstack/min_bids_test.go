//go:build !(unit && (windows || darwin))

package devstack

import (
	"context"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/devstack"

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
func (suite *MinBidsSuite) SetupSuite() {

}

// Before each test
func (suite *MinBidsSuite) SetupTest() {
	err := system.InitConfigForTesting()
	require.NoError(suite.T(), err)
}

func (suite *MinBidsSuite) TearDownTest() {
}

func (suite *MinBidsSuite) TearDownSuite() {

}

type minBidsTestCase struct {
	nodes          int
	shards         int
	concurrency    int
	minBids        int
	expectedResult map[model.JobStateType]int
	errorStates    []model.JobStateType
}

func (suite *MinBidsSuite) TestMinBids() {

	runTest := func(
		testCase minBidsTestCase,
	) {
		ctx := context.Background()
		t := system.GetTracer()
		ctx, span := system.NewRootSpan(ctx, t, "pkg/test/devstack/min_bids_test")
		defer span.End()

		stack, _ := SetupTest(
			ctx,
			suite.T(),
			testCase.nodes,
			0,
			false,
			computenode.NewDefaultComputeNodeConfig(),
		)

		dirPath, err := prepareFolderWithFiles(suite.T(), testCase.shards)
		require.NoError(suite.T(), err)

		directoryCid, err := devstack.AddFileToNodes(ctx, dirPath, devstack.ToIPFSClients(stack.Nodes[:testCase.nodes])...)
		require.NoError(suite.T(), err)

		apiUri := stack.Nodes[0].APIServer.GetURI()
		apiClient := publicapi.NewAPIClient(apiUri)

		j := &model.Job{}
		j.Spec = testutils.DockerRunJob()
		j.Spec.Inputs = []model.StorageSpec{
			{
				StorageSource: model.StorageSourceIPFS,
				CID:           directoryCid,
				Path:          "/input",
			},
		}
		j.Spec.Sharding = model.JobShardingConfig{
			GlobPattern: "/input/*",
			BatchSize:   1,
		}

		j.Deal = model.Deal{
			Concurrency: testCase.concurrency,
			MinBids:     testCase.minBids,
		}

		createdJob, err := apiClient.Submit(ctx, j, nil)
		require.NoError(suite.T(), err)
		resolver := apiClient.GetJobStateResolver()

		err = resolver.Wait(
			ctx,
			createdJob.ID,
			3,
			job.WaitThrowErrors(testCase.errorStates),
			job.WaitForJobStates(testCase.expectedResult),
		)
		require.NoError(suite.T(), err)

	}

	// sanity test that with min bids at zero and 1 node we get the job through
	suite.Run("minBids0And1Node", func() {
		runTest(minBidsTestCase{
			nodes:       1,
			shards:      1,
			concurrency: 1,
			minBids:     0,
			expectedResult: map[model.JobStateType]int{
				model.JobStateCompleted: 1,
			},
			errorStates: []model.JobStateType{
				model.JobStateError,
			},
		})
	})

	// test that when min bids is concurrency we get the job through
	suite.Run("minBidsIsConcurrency", func() {
		runTest(minBidsTestCase{
			nodes:       3,
			shards:      1,
			concurrency: 3,
			minBids:     3,
			expectedResult: map[model.JobStateType]int{
				model.JobStateCompleted: 3,
			},
			errorStates: []model.JobStateType{
				model.JobStateError,
			},
		})
	})

	// test that no bids are made because there are not enough nodes on the network
	// to satisfy the min bids
	suite.Run("noBids", func() {
		runTest(minBidsTestCase{
			nodes:       3,
			shards:      1,
			concurrency: 3,
			minBids:     5,
			expectedResult: map[model.JobStateType]int{
				model.JobStateBidding: 3,
			},
			errorStates: []model.JobStateType{
				model.JobStateError,
				model.JobStateWaiting,
				model.JobStateRunning,
				model.JobStateVerifying,
				model.JobStateCompleted,
			},
		})
	})

}
