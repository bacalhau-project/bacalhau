package devstack

import (
	"context"
	"io/ioutil"
	"path/filepath"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/devstack"

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

type DevStackSuite struct {
	suite.Suite
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestDevStackSuite(t *testing.T) {
	suite.Run(t, new(DevStackSuite))
}

// Before all suite
func (suite *DevStackSuite) SetupAllSuite() {

}

// Before each test
func (suite *DevStackSuite) SetupTest() {
	err := system.InitConfigForTesting()
	require.NoError(suite.T(), err)
}

func (suite *DevStackSuite) TearDownTest() {

}

func (suite *DevStackSuite) TearDownAllSuite() {

}

// re-use the docker executor tests but full end to end with libp2p transport
// and 3 nodes
func devStackDockerStorageTest(
	t *testing.T,
	testCase scenario.TestCase,
	nodeCount int,
) {
	ctx := context.Background()

	stack, cm := SetupTest(
		ctx,
		t,
		nodeCount,
		0,
		computenode.NewDefaultComputeNodeConfig(),
	)
	defer TeardownTest(stack, cm)

	nodeIDs, err := stack.GetNodeIds()
	require.NoError(t, err)

	inputStorageList, err := testCase.SetupStorage(ctx, model.StorageSourceIPFS, devstack.ToIPFSClients(stack.Nodes[:nodeCount])...)
	require.NoError(t, err)

	jobSpec := model.JobSpec{
		Engine:    model.EngineDocker,
		Verifier:  model.VerifierNoop,
		Publisher: model.PublisherIpfs,
		Docker:    testCase.GetJobSpec(),
		Inputs:    inputStorageList,
		Outputs:   testCase.Outputs,
	}

	jobDeal := model.JobDeal{
		Concurrency: nodeCount,
	}

	apiUri := stack.Nodes[0].APIServer.GetURI()
	apiClient := publicapi.NewAPIClient(apiUri)
	submittedJob, err := apiClient.Submit(ctx, jobSpec, jobDeal, nil)
	require.NoError(t, err)

	resolver := apiClient.GetJobStateResolver()

	err = resolver.Wait(
		ctx,
		submittedJob.ID,
		len(nodeIDs),
		job.WaitThrowErrors([]model.JobStateType{
			model.JobStateCancelled,
			model.JobStateError,
		}),
		job.WaitForJobStates(map[model.JobStateType]int{
			model.JobStateCompleted: len(nodeIDs),
		}),
	)
	require.NoError(t, err)

	shards, err := resolver.GetShards(ctx, submittedJob.ID)
	require.NoError(t, err)

	// now we check the actual results produced by the ipfs verifier
	for _, shard := range shards {
		node, err := stack.GetNode(ctx, shard.NodeID)
		require.NoError(t, err)

		outputDir, err := ioutil.TempDir("", "bacalhau-ipfs-devstack-test")
		require.NoError(t, err)
		require.NotEmpty(t, shard.PublishedResult.Cid)

		outputPath := filepath.Join(outputDir, shard.PublishedResult.Cid)
		err = node.IPFSClient.Get(ctx, shard.PublishedResult.Cid, outputPath)
		require.NoError(t, err)

		err = testCase.ResultsChecker(outputPath)
		require.NoError(t, err)
	}
}

func (suite *DevStackSuite) TestCatFileStdout() {
	devStackDockerStorageTest(
		suite.T(),
		scenario.CatFileToStdout(),
		3,
	)
}

func (suite *DevStackSuite) TestCatFileOutputVolume() {
	devStackDockerStorageTest(
		suite.T(),
		scenario.CatFileToVolume(),
		1,
	)
}

func (suite *DevStackSuite) TestGrepFile() {
	devStackDockerStorageTest(
		suite.T(),
		scenario.GrepFile(),
		3,
	)
}

func (suite *DevStackSuite) TestSedFile() {
	devStackDockerStorageTest(
		suite.T(),
		scenario.SedFile(),
		3,
	)
}

func (suite *DevStackSuite) TestAwkFile() {
	devStackDockerStorageTest(
		suite.T(),
		scenario.AwkFile(),
		3,
	)
}
