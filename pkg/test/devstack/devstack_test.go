package devstack

import (
	"context"
	"io/ioutil"
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

	inputStorageList, err := testCase.SetupStorage(stack, storage.IPFSAPICopy, nodeCount)
	require.NoError(t, err)

	jobSpec := &executor.JobSpec{
		Engine:   executor.EngineDocker,
		Verifier: verifier.VerifierIpfs,
		Docker:   testCase.GetJobSpec(),
		Inputs:   inputStorageList,
		Outputs:  testCase.Outputs,
	}

	jobDeal := &executor.JobDeal{
		Concurrency: nodeCount,
	}

	apiUri := stack.Nodes[0].APIServer.GetURI()
	apiClient := publicapi.NewAPIClient(apiUri)
	submittedJob, err := apiClient.Submit(ctx, jobSpec, jobDeal, nil)
	require.NoError(t, err)

	// wait for the job to complete across all nodes
	err = stack.WaitForJob(ctx, submittedJob.ID,
		devstack.WaitForJobThrowErrors([]executor.JobStateType{
			executor.JobStateBidRejected,
			executor.JobStateError,
		}),
		devstack.WaitForJobAllHaveState(nodeIDs, executor.JobStateComplete),
	)
	require.NoError(t, err)

	loadedJob, ok, err := apiClient.Get(ctx, submittedJob.ID)
	require.True(t, ok)
	require.NoError(t, err)

	// now we check the actual results produced by the ipfs verifier
	for nodeID, state := range loadedJob.State {
		node, err := stack.GetNode(ctx, nodeID)
		require.NoError(t, err)

		outputDir, err := ioutil.TempDir("", "bacalhau-ipfs-devstack-test")
		require.NoError(t, err)

		err = node.IpfsClient.Get(ctx, state.ResultsID, outputDir)
		require.NoError(t, err)

		testCase.ResultsChecker(outputDir + "/" + state.ResultsID)
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
