package devstack

import (
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/devstack"
	"github.com/filecoin-project/bacalhau/pkg/executor"
	ipfs_http "github.com/filecoin-project/bacalhau/pkg/ipfs/http"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/publicapi"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/test/scenario"
	"github.com/filecoin-project/bacalhau/pkg/types"
	"github.com/filecoin-project/bacalhau/pkg/verifier"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
)

var STORAGE_DRIVER_NAMES = []string{
	storage.IPFS_FUSE_DOCKER,
	storage.IPFS_API_COPY,
}

func SetupTest(
	t *testing.T,
	nodes int,
	badActors int,
) (*devstack.DevStack, *system.CancelContext) {
	cancelContext := system.GetCancelContextWithSignals()
	getExecutors := func(ipfsMultiAddress string, nodeIndex int) (map[string]executor.Executor, error) {
		return executor.NewDockerIPFSExecutors(cancelContext, ipfsMultiAddress, fmt.Sprintf("devstacknode%d", nodeIndex))
	}
	getVerifiers := func(ipfsMultiAddress string, nodeIndex int) (map[string]verifier.Verifier, error) {
		return verifier.NewIPFSVerifiers(cancelContext, ipfsMultiAddress)
	}
	stack, err := devstack.NewDevStack(
		cancelContext,
		nodes,
		badActors,
		getExecutors,
		getVerifiers,
	)
	assert.NoError(t, err)
	if err != nil {
		log.Fatal().Msg(fmt.Sprintf("Unable to create devstack: %s", err))
	}
	return stack, cancelContext
}

// this might be called multiple times if KEEP_STACK is active
// the first time - once the test has completed, this function will be called
// it will reset the KEEP_STACK variable so the user can ctrl+c the running stack
func TeardownTest(stack *devstack.DevStack, cancelContext *system.CancelContext) {
	if !system.ShouldKeepStack() {
		cancelContext.Stop()
	} else {
		stack.PrintNodeInfo()
		select {}
	}
}

// re-use the docker executor tests but full end to end with libp2p transport
// and 3 nodes
func devStackDockerStorageTest(
	t *testing.T,
	testCase scenario.TestCase,
	nodeCount int,
) {

	stack, cancelContext := SetupTest(
		t,
		nodeCount,
		0,
	)

	defer TeardownTest(stack, cancelContext)

	apiUri := stack.Nodes[0].ApiServer.GetURI()
	apiClient := publicapi.NewAPIClient(apiUri)

	inputStorageList, err := testCase.SetupStorage(stack, storage.IPFS_API_COPY, nodeCount)
	assert.NoError(t, err)

	jobSpec := &types.JobSpec{
		Engine:   string(executor.EXECUTOR_DOCKER),
		Verifier: string(verifier.VERIFIER_IPFS),
		Vm:       testCase.GetJobSpec(),
		Inputs:   inputStorageList,
		Outputs:  testCase.Outputs,
	}

	jobDeal := &types.JobDeal{
		Concurrency: nodeCount,
	}

	submittedJob, err := apiClient.Submit(jobSpec, jobDeal)
	assert.NoError(t, err)

	if err != nil {
		t.FailNow()
	}

	// wait for the job to complete across all nodes
	err = stack.WaitForJob(submittedJob.Id, map[string]int{
		system.JOB_STATE_COMPLETE: nodeCount,
	}, []string{
		system.JOB_STATE_BID_REJECTED,
		system.JOB_STATE_ERROR,
	})
	assert.NoError(t, err)

	loadedJob, err := apiClient.Get(submittedJob.Id)
	assert.NoError(t, err)

	// now we check the actual results produced by the ipfs verifier
	for nodeId, state := range loadedJob.State {
		node, err := stack.GetNode(nodeId)
		assert.NoError(t, err)
		outputDir, err := ioutil.TempDir("", "bacalhau-ipfs-devstack-test")
		ipfsClient, err := ipfs_http.NewIPFSHttpClient(cancelContext.Ctx, node.IpfsNode.ApiAddress())
		ipfsClient.DownloadTar(outputDir, state.ResultsId)
		testCase.ResultsChecker(outputDir + "/" + state.ResultsId)
	}
}

func TestCatFileStdout(t *testing.T) {
	devStackDockerStorageTest(
		t,
		scenario.CatFileToStdout(t),
		3,
	)
}

func TestCatFileOutputVolume(t *testing.T) {
	devStackDockerStorageTest(
		t,
		scenario.CatFileToVolume(t),
		1,
	)
}

func TestGrepFile(t *testing.T) {
	devStackDockerStorageTest(
		t,
		scenario.GrepFile(t),
		3,
	)
}

func TestSedFile(t *testing.T) {
	devStackDockerStorageTest(
		t,
		scenario.SedFile(t),
		3,
	)
}

func TestAwkFile(t *testing.T) {
	devStackDockerStorageTest(
		t,
		scenario.AwkFile(t),
		3,
	)
}
