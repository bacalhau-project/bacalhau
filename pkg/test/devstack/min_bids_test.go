package devstack

import (
	"context"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/requesternode"

	"github.com/filecoin-project/bacalhau/pkg/devstack"
	"github.com/filecoin-project/bacalhau/pkg/logger"

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

type MinBidsSuite struct {
	suite.Suite
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestMinBidsSuite(t *testing.T) {
	suite.Run(t, new(MinBidsSuite))
}

// Before each test
func (s *MinBidsSuite) SetupTest() {
	logger.ConfigureTestLogging(s.T())
	err := system.InitConfigForTesting()
	require.NoError(s.T(), err)
}

type minBidsTestCase struct {
	nodes          int
	shards         int
	concurrency    int
	minBids        int
	expectedResult map[model.JobStateType]int
	errorStates    []model.JobStateType
}

func (s *MinBidsSuite) testMinBids(testCase minBidsTestCase) {
	ctx := context.Background()
	t := system.GetTracer()
	ctx, span := system.NewRootSpan(ctx, t, "pkg/test/devstack/min_bids_test")
	defer span.End()

	stack, _ := SetupTest(
		ctx,
		s.T(),
		testCase.nodes,
		0,
		false,
		computenode.NewDefaultComputeNodeConfig(),
		requesternode.NewDefaultRequesterNodeConfig(),
	)

	dirPath, err := prepareFolderWithFiles(s.T(), testCase.shards)
	require.NoError(s.T(), err)

	directoryCid, err := devstack.AddFileToNodes(ctx, dirPath, devstack.ToIPFSClients(stack.Nodes[:testCase.nodes])...)
	require.NoError(s.T(), err)

	apiUri := stack.Nodes[0].APIServer.GetURI()
	apiClient := publicapi.NewAPIClient(apiUri)

	scn := scenario.WasmHelloWorld()
	j := &model.Job{}
	j.Spec = scn.GetJobSpec()
	j.Spec.Verifier = model.VerifierNoop
	j.Spec.Publisher = model.PublisherIpfs
	j.Spec.Contexts, err = scn.SetupContext(ctx, model.StorageSourceIPFS, devstack.ToIPFSClients(stack.Nodes[:testCase.nodes])...)
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
	require.NoError(s.T(), err)
	resolver := apiClient.GetJobStateResolver()

	err = resolver.Wait(
		ctx,
		createdJob.ID,
		3,
		job.WaitThrowErrors(testCase.errorStates),
		job.WaitForJobStates(testCase.expectedResult),
	)
	require.NoError(s.T(), err)

}

func (s *MinBidsSuite) TestMinBids_0and1Node() {
	// sanity test that with min bids at zero and 1 node we get the job through
	s.testMinBids(minBidsTestCase{
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
}

func (s *MinBidsSuite) TestMinBids_isConcurrency() {
	// test that when min bids is concurrency we get the job through
	s.testMinBids(minBidsTestCase{
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

}

func (s *MinBidsSuite) TestMinBids_noBids() {
	// test that no bids are made because there are not enough nodes on the network
	// to satisfy the min bids
	s.testMinBids(minBidsTestCase{
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

}
