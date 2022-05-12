package devstack

import (
	"fmt"
	"testing"

	"context"
	"time"

	"github.com/davecgh/go-spew/spew"
	"github.com/filecoin-project/bacalhau/pkg/devstack"
	"github.com/filecoin-project/bacalhau/pkg/executor"
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
) {

	// the inner test handler that is given the storage driver factory
	// and output mode that we are looping over internally
	runTest := func(
		storageDriver string,
	) {

		stack, cancelFunction := SetupTest(
			t,
			3,
			0,
		)

		defer TeardownTest(stack, cancelFunction)

		inputStorageList, err := setupStorage(stack, storageDriver)
		assert.NoError(t, err)

		// this is stdout mode
		outputs := []types.StorageSpec{}

		job := &types.Job{
			Id:    "test-job",
			Owner: "test-owner",
			Spec: &types.JobSpec{
				Engine:  executor.EXECUTOR_DOCKER,
				Vm:      getJobSpec(dockertests.OutputModeStdout),
				Inputs:  inputStorageList,
				Outputs: outputs,
			},
			Deal: &types.JobDeal{
				Concurrency:   3,
				AssignedNodes: []string{},
			},
		}

		spew.Dump(job)

		// isInstalled, err := dockerExecutor.IsInstalled()
		// assert.NoError(t, err)
		// assert.True(t, isInstalled)

		// for _, inputStorageSpec := range inputStorageList {
		// 	hasStorage, err := dockerExecutor.HasStorage(inputStorageSpec)
		// 	assert.NoError(t, err)
		// 	assert.True(t, hasStorage)
		// }

		// resultsDirectory, err := dockerExecutor.RunJob(job)
		// assert.NoError(t, err)

		// if err != nil {
		// 	t.FailNow()
		// }

		// checkResults(resultsDirectory, outputMode)
	}

	for _, storageDriverName := range STORAGE_DRIVER_NAMES {
		log.Debug().Msgf("Running test %s with storage driver %s", name, storageDriverName)
		runTest(storageDriverName)
	}
}
