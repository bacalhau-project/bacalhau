package scenario

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/devstack"
	"github.com/filecoin-project/bacalhau/pkg/executor"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/storage/ipfs/api_copy"
	"github.com/filecoin-project/bacalhau/pkg/storage/ipfs/fuse_docker"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
)

type TestCase struct {
	Name           string
	SetupStorage   ISetupStorage
	ResultsChecker ICheckResults
	GetJobSpec     IGetJobSpec
	Outputs        []storage.StorageSpec
}

type StorageDriverFactory struct {
	Name          string
	DriverFactory IGetStorageDriver
}

type IExpectedMode int

const (
	ExpectedModeEquals IExpectedMode = iota
	ExpectedModeContains
)

type IGetStorageDriver func(stack *devstack.DevStackIPFS) (storage.StorageProvider, error)
type ISetupStorage func(stack devstack.IDevStack, driverName string, nodeCount int) ([]storage.StorageSpec, error)
type ICheckResults func(resultsDir string)
type IGetJobSpec func() executor.JobSpecVM

/*

	Storage Drivers

*/
func FuseStorageDriverFactoryHandler(stack *devstack.DevStackIPFS) (storage.StorageProvider, error) {
	return fuse_docker.NewStorageProvider(
		stack.CleanupManager, stack.Nodes[0].IpfsNode.ApiAddress())
}

func APICopyStorageDriverFactoryHandler(stack *devstack.DevStackIPFS) (storage.StorageProvider, error) {
	return api_copy.NewStorageProvider(
		stack.CleanupManager, stack.Nodes[0].IpfsNode.ApiAddress())
}

var FuseStorageDriverFactory = StorageDriverFactory{
	Name:          "fuse",
	DriverFactory: FuseStorageDriverFactoryHandler,
}

var APICopyStorageDriverFactory = StorageDriverFactory{
	Name:          "apiCopy",
	DriverFactory: APICopyStorageDriverFactoryHandler,
}

var STORAGE_DRIVER_FACTORIES = []StorageDriverFactory{
	//	FuseStorageDriverFactory,
	APICopyStorageDriverFactory,
}

var STORAGE_DRIVER_FACTORIES_FUSE = []StorageDriverFactory{
	FuseStorageDriverFactory,
}

var STORAGE_DRIVER_FACTORIES_API_COPY = []StorageDriverFactory{
	APICopyStorageDriverFactory,
}

/*

	Setup storage

*/

func singleFileSetupStorageWithData(
	t *testing.T,
	fileContents string,
	mountPath string,
) ISetupStorage {
	return func(stack devstack.IDevStack, driverName string, nodeCount int) ([]storage.StorageSpec, error) {
		fileCid, err := stack.AddTextToNodes(nodeCount, []byte(fileContents))
		assert.NoError(t, err)
		inputStorageSpecs := []storage.StorageSpec{
			{
				Engine: driverName,
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
	return func(stack devstack.IDevStack, driverName string, nodeCount int) ([]storage.StorageSpec, error) {
		fileCid, err := stack.AddFileToNodes(nodeCount, filePath)
		assert.NoError(t, err)
		inputStorageSpecs := []storage.StorageSpec{
			{
				Engine: driverName,
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
	resultsDir string,
	filePath string,
) ([]byte, error) {
	outputFile := fmt.Sprintf("%s/%s", resultsDir, filePath)
	return os.ReadFile(outputFile)
}

func singleFileResultsChecker(
	t *testing.T,
	outputFilePath string,
	expectedString string,
	expectedMode IExpectedMode,
	expectedLines int,
) ICheckResults {
	return func(resultsDir string) {

		resultsContent, err := singleFileGetData(resultsDir, outputFilePath)
		assert.NoError(t, err)

		log.Trace().Msgf("test checking: %s/%s resultsContent: %s", resultsDir, outputFilePath, resultsContent)

		actualLineCount := len(strings.Split(string(resultsContent), "\n"))
		assert.Equal(t, expectedLines, actualLineCount, fmt.Sprintf("Count mismatch:\nExpected: %d\nActual: %d", expectedLines, actualLineCount))

		if expectedMode == ExpectedModeEquals {
			assert.Equal(t, expectedString, string(resultsContent))
		} else if expectedMode == ExpectedModeContains {
			assert.True(t, strings.Contains(string(resultsContent), expectedString))
		} else {
			t.Fail()
		}
	}
}
