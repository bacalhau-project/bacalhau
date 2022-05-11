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
const TEST_OUTPUT_VOLUME_NAME = "output_volume"
const TEST_OUTPUT_VOLUME_MOUNT_PATH = "/output_volume"

type IGetStorageDriver func(stack *devstack.DevStack_IPFS) (storage.StorageProvider, error)
type ISetupStorage func(stack *devstack.DevStack_IPFS) ([]types.StorageSpec, error)
type ICheckResults func(resultsDir string, outputMode IOutputMode)
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

var STORAGE_DRIVER_FACTORIES = []struct {
	name          string
	driverFactory IGetStorageDriver
}{
	{
		name:          "fuse",
		driverFactory: fuseStorageDriverFactory,
	},
	{
		name:          "apiCopy",
		driverFactory: apiCopyStorageDriverFactory,
	},
}

var OUTPUT_MODES = []IOutputMode{
	OutputModeStdout,
	OutputModeVolume,
}

/*

	Setup storage

*/

func singleFileSetupStorageWithData(
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

func singleFileSetupStorageWithFile(
	t *testing.T,
	filePath string,
	mountPath string,
) ISetupStorage {
	return func(stack *devstack.DevStack_IPFS) ([]types.StorageSpec, error) {
		fileCid, err := stack.AddFileToNodes(1, filePath)
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

/*

	Results checkers

*/

func singleFileGetData(
	outputMode IOutputMode,
	resultsDir string,
	outputPathVolume string,
) ([]byte, error) {
	outputPath := "stdout"
	if outputMode == OutputModeVolume {
		outputPath = fmt.Sprintf("%s/%s", TEST_OUTPUT_VOLUME_NAME, outputPathVolume)
	}
	outputFile := fmt.Sprintf("%s/%s", resultsDir, outputPath)
	return os.ReadFile(outputFile)
}

func singleFileResultsCheckerContains(
	t *testing.T,
	outputPathVolume string,
	expectedString string,
	expectedMode IExpectedMode,
	expectedLines int,
) ICheckResults {
	return func(resultsDir string, outputMode IOutputMode) {
		resultsContent, err := singleFileGetData(outputMode, resultsDir, outputPathVolume)
		assert.NoError(t, err)

		log.Debug().Msgf("resultsContent: %s", resultsContent)

		actual_line_count := len(strings.Split(string(resultsContent), "\n"))
		assert.Equal(t, expectedLines, actual_line_count, fmt.Sprintf("Count mismatch:\nExpected: %d\nActual: %d", expectedLines, actual_line_count))

		if expectedMode == ExpectedModeEquals {
			assert.Equal(t, expectedString, string(resultsContent))
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
	name string,
	setupStorage ISetupStorage,
	checkResults ICheckResults,
	getJobSpec IGetJobSpec,
) {

	// the inner test handler that is given the storage driver factory
	// and output mode that we are looping over internally
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

		// this is stdout mode
		outputs := []types.StorageSpec{}

		// in this mode we mount an output volume to collect the results
		if outputMode == OutputModeVolume {
			outputs = []types.StorageSpec{
				{
					Name: TEST_OUTPUT_VOLUME_NAME,
					Path: TEST_OUTPUT_VOLUME_MOUNT_PATH,
				},
			}
		}

		job := &types.Job{
			Id:    "test-job",
			Owner: "test-owner",
			Spec: &types.JobSpec{
				Engine:  executor.EXECUTOR_DOCKER,
				Vm:      getJobSpec(outputMode),
				Inputs:  inputStorageList,
				Outputs: outputs,
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

		if err != nil {
			t.FailNow()
		}

		checkResults(resultsDirectory, outputMode)
	}

	for _, storageDriverFactory := range STORAGE_DRIVER_FACTORIES {
		for _, outputMode := range OUTPUT_MODES {
			log.Debug().Msgf("Running test %s with storage driver %s and output mode %d", name, storageDriverFactory.name, outputMode)
			runTest(storageDriverFactory.driverFactory, outputMode)
		}
	}
}

/*

	Utils

*/

// if we are running a stdout based test - then leave the entrypoint alone
// otherwise - we want to redirect the output of the job to an output volume
func convertEntryPoint(outputMode IOutputMode, appendPath string, cmds []string) []string {
	// this means we are in stdout mode
	if outputMode == OutputModeStdout {
		return cmds
	}
	outputFile := TEST_OUTPUT_VOLUME_MOUNT_PATH
	if appendPath != "" {
		outputFile = fmt.Sprintf("%s/%s", TEST_OUTPUT_VOLUME_MOUNT_PATH, appendPath)
	}
	return []string{
		"bash",
		"-c",
		fmt.Sprintf("%s > %s", strings.Join(cmds, " "), outputFile),
	}
}
