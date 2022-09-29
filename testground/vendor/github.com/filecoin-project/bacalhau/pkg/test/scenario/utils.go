package scenario

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/devstack"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	apicopy "github.com/filecoin-project/bacalhau/pkg/storage/ipfs_apicopy"
	fusedocker "github.com/filecoin-project/bacalhau/pkg/storage/ipfs_fusedocker"
	"github.com/rs/zerolog/log"
	"github.com/stretchr/testify/require"
)

type TestCase struct {
	Name           string
	SetupStorage   ISetupStorage
	ResultsChecker ICheckResults
	GetJobSpec     IGetJobSpec
	Outputs        []model.StorageSpec
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

type IGetStorageDriver func(ctx context.Context, stack *devstack.DevStackIPFS) (storage.StorageProvider, error)

//nolint:lll
type ISetupStorage func(ctx context.Context, stack devstack.IDevStack, driverName model.StorageSourceType, nodeCount int) ([]model.StorageSpec, error)
type ICheckResults func(resultsDir string)
type IGetJobSpec func() model.JobSpecDocker

/*
Storage Drivers
*/
func FuseStorageDriverFactoryHandler(ctx context.Context, stack *devstack.DevStackIPFS) (storage.StorageProvider, error) {
	return fusedocker.NewStorageProvider(
		ctx, stack.CleanupManager, stack.IPFSClients[0].APIAddress())
}

func APICopyStorageDriverFactoryHandler(ctx context.Context, stack *devstack.DevStackIPFS) (storage.StorageProvider, error) {
	return apicopy.NewStorageProvider(
		stack.CleanupManager, stack.IPFSClients[0].APIAddress())
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
	ctx context.Context,
	fileContents string,
	mountPath string,
) ISetupStorage {
	//nolint:lll
	return func(ctx context.Context, stack devstack.IDevStack, driverName model.StorageSourceType, nodeCount int) ([]model.StorageSpec, error) {
		fileCid, err := stack.AddTextToNodes(ctx, nodeCount, []byte(fileContents))
		require.NoError(t, err)
		inputStorageSpecs := []model.StorageSpec{
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
	ctx context.Context,
	filePath string,
	mountPath string,
) ISetupStorage {
	//nolint:lll
	return func(ctx context.Context, stack devstack.IDevStack, driverName model.StorageSourceType, nodeCount int) ([]model.StorageSpec, error) {
		fileCid, err := stack.AddFileToNodes(ctx, nodeCount, filePath)
		require.NoError(t, err)
		inputStorageSpecs := []model.StorageSpec{
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
	ctx context.Context,
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
