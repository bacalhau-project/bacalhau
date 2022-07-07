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
	"github.com/filecoin-project/bacalhau/pkg/storage/ipfs/apicopy"
	"github.com/filecoin-project/bacalhau/pkg/storage/ipfs/fusedocker"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/require"
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
type IGetJobSpec func() executor.JobSpecDocker

/*

	Storage Drivers

*/
func FuseStorageDriverFactoryHandler(stack *devstack.DevStackIPFS) (storage.StorageProvider, error) {
	return fusedocker.NewStorageProvider(
		stack.CleanupManager, stack.Nodes[0].IpfsClient.APIAddress())
}

func APICopyStorageDriverFactoryHandler(stack *devstack.DevStackIPFS) (storage.StorageProvider, error) {
	return apicopy.NewStorageProvider(
		stack.CleanupManager, stack.Nodes[0].IpfsClient.APIAddress())
}

var FuseStorageDriverFactory = StorageDriverFactory{
	Name:          "fuse",
	DriverFactory: FuseStorageDriverFactoryHandler,
}

var APICopyStorageDriverFactory = StorageDriverFactory{
	Name:          "apiCopy",
	DriverFactory: APICopyStorageDriverFactoryHandler,
}

var StorageDriverFactories = []StorageDriverFactory{
	//	FuseStorageDriverFactory,
	APICopyStorageDriverFactory,
}

var StorageDriverFactoriesFuse = []StorageDriverFactory{
	FuseStorageDriverFactory,
}

var StorageDriverFactoriesAPICopy = []StorageDriverFactory{
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
		require.NoError(t, err)
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
		require.NoError(t, err)
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
		require.NoError(t, err)

		log.Trace().Msgf("test checking: %s/%s resultsContent: %s", resultsDir, outputFilePath, resultsContent)

		actualLineCount := len(strings.Split(string(resultsContent), "\n"))
		require.Equal(t, expectedLines, actualLineCount, fmt.Sprintf("Count mismatch:\nExpected: %d\nActual: %d", expectedLines, actualLineCount))

		if expectedMode == ExpectedModeEquals {
			require.Equal(t, expectedString, string(resultsContent))
		} else if expectedMode == ExpectedModeContains {
			require.True(t, strings.Contains(string(resultsContent), expectedString))
		} else {
			t.Fail()
		}
	}
}
