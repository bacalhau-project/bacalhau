package docker

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/devstack"
	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/executor/docker"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/storage/ipfs/api_copy"
	"github.com/filecoin-project/bacalhau/pkg/storage/ipfs/fuse_docker"
	"github.com/filecoin-project/bacalhau/pkg/test/ipfs"
	"github.com/filecoin-project/bacalhau/pkg/types"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
)

type IExpectedMode int

const (
	ExpectedModeEquals IExpectedMode = iota
	ExpectedModeContains
)

type IOutputMode int

const (
	OutputModeStdout IOutputMode = iota
	OutputModeVolume
)

const TEST_STORAGE_DRIVER_NAME = "testdriver"

type IGetStorageDriver func(stack *devstack.DevStack_IPFS) (storage.StorageProvider, error)
type ISetupStorage func(stack *devstack.DevStack_IPFS) ([]types.StorageSpec, error)
type ICheckResults func(resultsDir, resultsPath string)
type IGetJobSpec func(outputMode IOutputMode) types.JobSpecVm

/*

	Storage Drivers

*/
func fuseStorageDriverFactory(stack *devstack.DevStack_IPFS) (storage.StorageProvider, error) {
	return fuse_docker.NewIpfsFuseDocker(stack.Ctx, stack.Nodes[0].IpfsNode.ApiAddress())
}

func apiCopyStorageDriverFactory(stack *devstack.DevStack_IPFS) (storage.StorageProvider, error) {
	return api_copy.NewIpfsApiCopy(stack.Ctx, stack.Nodes[0].IpfsNode.ApiAddress())
}

var STORAGE_DRIVER_FACTORIES = []IGetStorageDriver{
	fuseStorageDriverFactory,
	apiCopyStorageDriverFactory,
}

var OUTPUT_MODES = []IOutputMode{
	OutputModeStdout,
	OutputModeVolume,
}

/*

	Results Checkers

*/

func singleFileSetupStorage(
	t *testing.T,
	fileContents string,
	mountPath string,
) ISetupStorage {
	return func(stack *devstack.DevStack_IPFS) ([]types.StorageSpec, error) {
		fileCid, err := stack.AddTextToNodes(1, []byte(fileContents))
		assert.NoError(t, err)
		inputStorageSpecs := []types.StorageSpec{
			{
				Engine: TEST_STORAGE_DRIVER_NAME,
				Cid:    fileCid,
				Path:   mountPath,
			},
		}
		return inputStorageSpecs, nil
	}

}

func singleFileResultsChecker(
	t *testing.T,
	expectedString string,
	expectedMode IExpectedMode,
) ICheckResults {
	return func(resultsDirectory, resultsPath string) {
		resultsContent, err := os.ReadFile(fmt.Sprintf("%s/%s", resultsDirectory, resultsPath))
		assert.NoError(t, err)

		log.Debug().Msgf("resultsContent: %s", resultsContent)

		if expectedMode == ExpectedModeEquals {
			assert.Equal(t, string(resultsContent), expectedString)
		} else if expectedMode == ExpectedModeContains {
			assert.True(t, strings.Contains(string(resultsContent), expectedString))
		} else {
			t.Fail()
		}
	}
}

/*

	Executor Job Tests

*/

// for each test we iterate over:
//  * storage drivers (ipfs_fuse, ipfs_api_copy)
//  * output types (stdout, named volume)

func dockerExecutorStorageTest(
	t *testing.T,
	setupStorage ISetupStorage,
	checkResults ICheckResults,
	getJobSpec IGetJobSpec,
) {

	runTest := func(
		getStorageDriver IGetStorageDriver,
		outputMode IOutputMode,
	) {
		stack, cancelFunction := ipfs.SetupTest(
			t,
			1,
		)

		defer ipfs.TeardownTest(stack, cancelFunction)

		storageDriver, err := getStorageDriver(stack)
		assert.NoError(t, err)

		dockerExecutor, err := docker.NewDockerExecutor(stack.Ctx, "dockertest", map[string]storage.StorageProvider{
			TEST_STORAGE_DRIVER_NAME: storageDriver,
		})
		assert.NoError(t, err)

		inputStorageList, err := setupStorage(stack)
		assert.NoError(t, err)

		job := &types.Job{
			Id:    "test-job",
			Owner: "test-owner",
			Spec: &types.JobSpec{
				Engine: executor.EXECUTOR_DOCKER,
				Vm:     getJobSpec(outputMode),
				Inputs: inputStorageList,
			},
			Deal: &types.JobDeal{
				Concurrency:   1,
				AssignedNodes: []string{},
			},
		}

		isInstalled, err := dockerExecutor.IsInstalled()
		assert.NoError(t, err)
		assert.True(t, isInstalled)

		for _, inputStorageSpec := range inputStorageList {
			hasStorage, err := dockerExecutor.HasStorage(inputStorageSpec)
			assert.NoError(t, err)
			assert.True(t, hasStorage)
		}

		resultsDirectory, err := dockerExecutor.RunJob(job)
		assert.NoError(t, err)

		checkResults(resultsDirectory, "stdout")
	}

	for _, getStorageDriver := range STORAGE_DRIVER_FACTORIES {
		runTest(getStorageDriver, OutputModeStdout)
	}
}
