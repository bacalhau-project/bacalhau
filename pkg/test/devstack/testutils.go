package devstack

import (
	"fmt"
	"testing"

	"context"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/filecoin-project/bacalhau/pkg/devstack"
	"github.com/filecoin-project/bacalhau/pkg/executor"
	jobutils "github.com/filecoin-project/bacalhau/pkg/job"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/system"
	dockertests "github.com/filecoin-project/bacalhau/pkg/test/docker"
	"github.com/filecoin-project/bacalhau/pkg/types"
	"github.com/stretchr/testify/assert"

	"github.com/rs/zerolog/log"
)

var STORAGE_DRIVER_NAMES = []string{
	storage.IPFS_FUSE_DOCKER,
	storage.IPFS_API_COPY,
}

func SetupTest(
	t *testing.T,
	nodes int,
	badActors int,
) (*devstack.DevStack, context.CancelFunc) {
	ctx, cancelFunction := system.GetCancelContext()

	getExecutors := func(ipfsMultiAddress string, nodeIndex int) (map[string]executor.Executor, error) {
		return devstack.NewDockerIPFSExecutors(ctx, ipfsMultiAddress, fmt.Sprintf("devstacknode%d", nodeIndex))
	}

	stack, err := devstack.NewDevStack(
		ctx,
		nodes,
		badActors,
		getExecutors,
	)
	assert.NoError(t, err)
	if err != nil {
		log.Fatal().Msg(fmt.Sprintf("Unable to create devstack: %s", err))
	}
	// TODO: add a waitgroup with checks on each part of a node
	// (i.e. libp2p connected, jsonrpc serving, ipfs functional)
	time.Sleep(time.Second * 2)
	return stack, cancelFunction
}

// this might be called multiple times if KEEP_STACK is active
// the first time - once the test has completed, this function will be called
// it will reset the KEEP_STACK variable so the user can ctrl+c the running stack
func TeardownTest(stack *devstack.DevStack, cancelFunction context.CancelFunc) {
	if !system.ShouldKeepStack() {
		cancelFunction()
		// need some time to let ipfs processes shut down
		time.Sleep(time.Second * 2)
	} else {
		stack.PrintNodeInfo()
		system.ClearKeepStack()
		select {}
	}
}

// re-use the docker executor tests but full end to end with libp2p transport
// and 3 nodes
func devStackDockerStorageTest(
	t *testing.T,
	name string,
	setupStorage dockertests.ISetupStorage,
	checkResults dockertests.ICheckResults,
	getJobSpec dockertests.IGetJobSpec,
	nodeCount int,
) {

	stack, cancelFunction := SetupTest(
		t,
		nodeCount,
		0,
	)

	defer TeardownTest(stack, cancelFunction)

	inputStorageList, err := setupStorage(stack, storage.IPFS_API_COPY, nodeCount)
	assert.NoError(t, err)

	// this is stdout mode
	outputs := []types.StorageSpec{}

	jobSpec := &types.JobSpec{
		Engine:  executor.EXECUTOR_DOCKER,
		Vm:      getJobSpec(dockertests.OutputModeStdout),
		Inputs:  inputStorageList,
		Outputs: outputs,
	}

	jobDeal := &types.JobDeal{
		Concurrency: nodeCount,
	}

	job, err := jobutils.SubmitJob(jobSpec, jobDeal, "127.0.0.1", stack.Nodes[0].JSONRpcNode.Port)
	assert.NoError(t, err)

	spew.Dump(job)

	err = stack.WaitForJob(job.Id, map[string]int{
		system.JOB_STATE_COMPLETE: nodeCount,
	}, []string{
		system.JOB_STATE_BID_REJECTED,
		system.JOB_STATE_ERROR,
	})
	assert.NoError(t, err)
}
