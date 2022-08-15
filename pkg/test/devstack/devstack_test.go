package devstack

import (
	"context"
	"io/ioutil"
	"path/filepath"
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
	"go.opentelemetry.io/otel/trace"
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
	system.InitConfigForTesting(suite.T())
}

func (suite *DevStackSuite) TearDownTest() {
}

func (suite *DevStackSuite) TearDownAllSuite() {

}

func newSpan(name string) (context.Context, trace.Span) {
	return system.Span(context.Background(), "devstack_test", name)
}

// re-use the docker executor tests but full end to end with libp2p transport
// and 3 nodes
func devStackDockerStorageTest(
	t *testing.T,
	testCase scenario.TestCase,
	nodeCount int,
) {
	ctx, span := newSpan(testCase.Name)
	defer span.End()

	stack, cm := SetupTest(
		t,
		nodeCount,
		0,
		computenode.NewDefaultComputeNodeConfig(),
	)
	defer TeardownTest(stack, cm)

	nodeIDs, err := stack.GetNodeIds()
	require.NoError(t, err)

	inputStorageList, err := testCase.SetupStorage(stack, storage.StorageSourceIPFS, nodeCount)
	require.NoError(t, err)

	jobSpec := executor.JobSpec{
		Engine:   executor.EngineDocker,
		Verifier: verifier.VerifierNoop,
		Docker:   testCase.GetJobSpec(),
		Inputs:   inputStorageList,
		Outputs:  testCase.Outputs,
	}

	jobDeal := executor.JobDeal{
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
		job.WaitThrowErrors([]executor.JobStateType{
			executor.JobStateCancelled,
			executor.JobStateError,
		}),
		job.WaitForJobStates(map[executor.JobStateType]int{
			executor.JobStateShardComplete: len(nodeIDs),
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

		outputPath := filepath.Join(outputDir, string(shard.ResultsProposal))
		err = node.IpfsClient.Get(ctx, string(shard.ResultsProposal), outputPath)
		require.NoError(t, err)

		testCase.ResultsChecker(outputPath)
	}
}

func (suite *DevStackSuite) TestCatFileStdout() {
	devStackDockerStorageTest(
		suite.T(),
		scenario.CatFileToStdout(suite.T()),
		3,
	)
}

func (suite *DevStackSuite) TestCatFileOutputVolume() {
	devStackDockerStorageTest(
		suite.T(),
		scenario.CatFileToVolume(suite.T()),
		1,
	)
}

func (suite *DevStackSuite) TestGrepFile() {
	devStackDockerStorageTest(
		suite.T(),
		scenario.GrepFile(suite.T()),
		3,
	)
}

func (suite *DevStackSuite) TestSedFile() {
	devStackDockerStorageTest(
		suite.T(),
		scenario.SedFile(suite.T()),
		3,
	)
}

func (suite *DevStackSuite) TestAwkFile() {
	devStackDockerStorageTest(
		suite.T(),
		scenario.AwkFile(suite.T()),
		3,
	)
}
