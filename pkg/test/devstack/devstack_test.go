package devstack

import (
	"context"
	"io/ioutil"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/compute_node"
	"github.com/filecoin-project/bacalhau/pkg/executor"
	ipfs_http "github.com/filecoin-project/bacalhau/pkg/ipfs/http"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/publicapi"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/test/scenario"
	"github.com/filecoin-project/bacalhau/pkg/types"
	"github.com/filecoin-project/bacalhau/pkg/verifier"
	"github.com/stretchr/testify/assert"
)

// re-use the docker executor tests but full end to end with libp2p transport
// and 3 nodes
func devStackDockerStorageTest(
	t *testing.T,
	testCase scenario.TestCase,
	nodeCount int,
) {
	ctx := context.Background()
	stack, cm := SetupTest(
		t,
		nodeCount,
		0,
		compute_node.NewDefaultJobSelectionPolicy(),
	)
	defer TeardownTest(stack, cm)

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

	apiUri := stack.Nodes[0].ApiServer.GetURI()
	apiClient := publicapi.NewAPIClient(apiUri)
	submittedJob, err := apiClient.Submit(jobSpec, jobDeal)
	assert.NoError(t, err)

	// wait for the job to complete across all nodes
	err = stack.WaitForJob(submittedJob.Id, map[string]int{
		system.JOB_STATE_COMPLETE: nodeCount,
	}, []string{
		system.JOB_STATE_BID_REJECTED,
		system.JOB_STATE_ERROR,
	})
	assert.NoError(t, err)

	loadedJob, ok, err := apiClient.Get(submittedJob.Id)
	assert.True(t, ok)
	assert.NoError(t, err)

	// now we check the actual results produced by the ipfs verifier
	for nodeId, state := range loadedJob.State {
		node, err := stack.GetNode(nodeId)
		assert.NoError(t, err)

		outputDir, err := ioutil.TempDir("", "bacalhau-ipfs-devstack-test")
		assert.NoError(t, err)

		ipfsClient, err := ipfs_http.NewIPFSHttpClient(
			node.IpfsNode.ApiAddress())
		assert.NoError(t, err)

		ipfsClient.DownloadTar(ctx, outputDir, state.ResultsId)
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
