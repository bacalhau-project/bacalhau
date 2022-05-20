package scenario

import (
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/devstack"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/storage/ipfs/api_copy"
	"github.com/filecoin-project/bacalhau/pkg/storage/ipfs/fuse_docker"
	"github.com/filecoin-project/bacalhau/pkg/types"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/assert"
)

type TestCase struct {
	Name           string
	SetupStorage   ISetupStorage
	ResultsChecker ICheckResults
	GetJobSpec     IGetJobSpec
	Outputs        []types.StorageSpec
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

type IGetStorageDriver func(stack *devstack.DevStack_IPFS) (storage.StorageProvider, error)
type ISetupStorage func(stack devstack.IDevStack, driverName string, nodeCount int) ([]types.StorageSpec, error)
type ICheckResults func(resultsDir string)
type IGetJobSpec func() types.JobSpecVm

/*

	Storage Drivers

*/
func FuseStorageDriverFactoryHandler(stack *devstack.DevStack_IPFS) (storage.StorageProvider, error) {
	return fuse_docker.NewIpfsFuseDocker(stack.CancelContext, stack.Nodes[0].IpfsNode.ApiAddress())
}

func ApiCopyStorageDriverFactoryHandler(stack *devstack.DevStack_IPFS) (storage.StorageProvider, error) {
	return api_copy.NewIpfsApiCopy(stack.CancelContext, stack.Nodes[0].IpfsNode.ApiAddress())
}

var FuseStorageDriverFactory = StorageDriverFactory{
	Name:          "fuse",
	DriverFactory: FuseStorageDriverFactoryHandler,
}

var ApiCopyStorageDriverFactory = StorageDriverFactory{
	Name:          "apiCopy",
	DriverFactory: ApiCopyStorageDriverFactoryHandler,
}

var STORAGE_DRIVER_FACTORIES = []StorageDriverFactory{
	FuseStorageDriverFactory,
	ApiCopyStorageDriverFactory,
}

var STORAGE_DRIVER_FACTORIES_FUSE = []StorageDriverFactory{
	FuseStorageDriverFactory,
}

var STORAGE_DRIVER_FACTORIES_API_COPY = []StorageDriverFactory{
	ApiCopyStorageDriverFactory,
}

/*

	Setup storage

*/

func singleFileSetupStorageWithData(
	t *testing.T,
	fileContents string,
	mountPath string,
) ISetupStorage {
	return func(stack devstack.IDevStack, driverName string, nodeCount int) ([]types.StorageSpec, error) {
		fileCid, err := stack.AddTextToNodes(nodeCount, []byte(fileContents))
		assert.NoError(t, err)
		inputStorageSpecs := []types.StorageSpec{
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
	return func(stack devstack.IDevStack, driverName string, nodeCount int) ([]types.StorageSpec, error) {
		fileCid, err := stack.AddFileToNodes(nodeCount, filePath)
		assert.NoError(t, err)
		inputStorageSpecs := []types.StorageSpec{
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

		log.Debug().Msgf("test checking: %s/%s resultsContent: %s", resultsDir, outputFilePath, resultsContent)

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
