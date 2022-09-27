package scenario

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/filecoin-project/bacalhau/pkg/ipfs"

	"github.com/filecoin-project/bacalhau/pkg/devstack"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	apicopy "github.com/filecoin-project/bacalhau/pkg/storage/ipfs_apicopy"
	fusedocker "github.com/filecoin-project/bacalhau/pkg/storage/ipfs_fusedocker"
	"github.com/rs/zerolog/log"
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
type ISetupStorage func(ctx context.Context, driverName model.StorageSourceType, ipfsClients ...*ipfs.Client) ([]model.StorageSpec, error)
type ICheckResults func(resultsDir string) error
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
	ctx context.Context,
	fileContents string,
	mountPath string,
) ISetupStorage {
	//nolint:lll
	return func(ctx context.Context, driverName model.StorageSourceType, clients ...*ipfs.Client) ([]model.StorageSpec, error) {
		fileCid, err := devstack.AddTextToNodes(ctx, []byte(fileContents), clients...)
		if err != nil {
			return nil, err
		}
		inputStorageSpecs := []model.StorageSpec{
			{
				Engine: driverName,
				Cid:    fileCid,
				Path:   mountPath,
			},
		}
		log.Debug().Msgf("Added file with cid %s", fileCid)
		return inputStorageSpecs, nil
	}
}

func singleFileSetupStorageWithFile(
	ctx context.Context,
	filePath string,
	mountPath string,
) ISetupStorage {
	//nolint:lll
	return func(ctx context.Context, driverName model.StorageSourceType, clients ...*ipfs.Client) ([]model.StorageSpec, error) {
		fileCid, err := devstack.AddFileToNodes(ctx, filePath, clients...)
		if err != nil {
			return nil, err
		}
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
	outputFile := filepath.Join(resultsDir, filePath)
	return os.ReadFile(outputFile)
}

func singleFileResultsChecker(
	ctx context.Context,
	outputFilePath string,
	expectedString string,
	expectedMode IExpectedMode,
	expectedLines int,
) ICheckResults {
	return func(resultsDir string) error {
		resultsContent, err := singleFileGetData(resultsDir, outputFilePath)
		if err != nil {
			return err
		}

		log.Debug().Msgf("test checking: %s/%s resultsContent: %s", resultsDir, outputFilePath, resultsContent)

		actualLineCount := len(strings.Split(string(resultsContent), "\n"))
		if actualLineCount != expectedLines {
			return fmt.Errorf("count mismatch:\nExpected: %d\nActual: %d", expectedLines, actualLineCount)
		}

		if expectedMode == ExpectedModeEquals {
			if string(resultsContent) != expectedString {
				return fmt.Errorf("content mismatch:\nExpected: %s\nActual: %s", expectedString, resultsContent)
			}
		} else if expectedMode == ExpectedModeContains {
			if !strings.Contains(string(resultsContent), expectedString) {
				return fmt.Errorf("content mismatch:\nExpected Contains: %s\nActual: %s", expectedString, resultsContent)
			}
		} else {
			return fmt.Errorf("unknown expected mode: %d", expectedMode)
		}
		return nil
	}
}
