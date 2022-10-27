//go:build !(unit && (windows || darwin))

package devstack

import (
	"context"
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
func (suite *DevStackSuite) SetupSuite() {

}

// Before each test
func (suite *DevStackSuite) SetupTest() {
	err := system.InitConfigForTesting()
	require.NoError(suite.T(), err)
}

func (suite *DevStackSuite) TearDownTest() {

}

func (suite *DevStackSuite) TearDownSuite() {

}

// re-use the docker executor tests but full end to end with libp2p transport
// and 3 nodes
func devStackDockerStorageTest(
	t *testing.T,
	testCase scenario.TestCase,
	nodeCount int,
) {
	ctx := context.Background()

	stack, _ := SetupTest(
		ctx,
		t,
		nodeCount,
		0,
		false,
		computenode.NewDefaultComputeNodeConfig(),
	)

	nodeIDs, err := stack.GetNodeIds()
	require.NoError(t, err)

	inputStorageList, err := testCase.SetupStorage(ctx, model.StorageSourceIPFS, devstack.ToIPFSClients(stack.Nodes[:nodeCount])...)
	require.NoError(t, err)

	j := &model.Job{}
	j.Spec = testCase.GetJobSpec()
	j.Spec.Verifier = model.VerifierNoop
	j.Spec.Publisher = model.PublisherIpfs
	j.Spec.Inputs = inputStorageList
	j.Spec.Outputs = testCase.Outputs
	j.Deal = model.Deal{
		Concurrency: nodeCount,
	}

	apiUri := stack.Nodes[0].APIServer.GetURI()
	apiClient := publicapi.NewAPIClient(apiUri)
	submittedJob, err := apiClient.Submit(ctx, j, nil)
	require.NoError(t, err)

	resolver := apiClient.GetJobStateResolver()

	err = resolver.Wait(
		ctx,
		submittedJob.ID,
		len(nodeIDs),
		job.WaitThrowErrors([]model.JobStateType{
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

		outputDir := t.TempDir()
		require.NotEmpty(t, shard.PublishedResult.CID)

		outputPath := filepath.Join(outputDir, shard.PublishedResult.CID)
		err = node.IPFSClient.Get(ctx, shard.PublishedResult.CID, outputPath)
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
