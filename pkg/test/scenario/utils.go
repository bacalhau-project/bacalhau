package scenario

import (
	"context"
	"fmt"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/filecoin-project/bacalhau/pkg/ipfs"
	"github.com/filecoin-project/bacalhau/pkg/job"

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
	Inputs         ISetupStorage
	Contexts       ISetupStorage
	Outputs        []model.StorageSpec
	Spec           model.Spec
	Deal           model.Deal
	ResultsChecker ICheckResults
	JobCheckers    []job.CheckStatesFunction
}

type StorageDriverFactory struct {
	Name          string
	DriverFactory IGetStorageDriver
}

type IGetStorageDriver func(ctx context.Context, stack *devstack.DevStackIPFS) (storage.Storage, error)

//nolint:lll
type ISetupStorage func(ctx context.Context, driverName model.StorageSourceType, ipfsClients ...*ipfs.Client) ([]model.StorageSpec, error)
type ICheckResults func(resultsDir string) error

/*
Storage Drivers
*/
func FuseStorageDriverFactoryHandler(ctx context.Context, stack *devstack.DevStackIPFS) (storage.Storage, error) {
	return fusedocker.NewStorageProvider(
		ctx, stack.CleanupManager, stack.IPFSClients[0].APIAddress())
}

func APICopyStorageDriverFactoryHandler(ctx context.Context, stack *devstack.DevStackIPFS) (storage.Storage, error) {
	return apicopy.NewStorage(
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

func StoredText(
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
				StorageSource: driverName,
				CID:           fileCid,
				Path:          mountPath,
			},
		}
		log.Debug().Msgf("Added file with cid %s", fileCid)
		return inputStorageSpecs, nil
	}
}

func StoredFile(
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
				StorageSource: driverName,
				CID:           fileCid,
				Path:          mountPath,
			},
		}
		return inputStorageSpecs, nil
	}
}

func URLDownload(
	server *httptest.Server,
	urlPath string,
	mountPath string,
) ISetupStorage {
	return func(ctx context.Context, driverName model.StorageSourceType, ipfsClients ...*ipfs.Client) ([]model.StorageSpec, error) {
		finalURL, err := url.JoinPath(server.URL, urlPath)
		return []model.StorageSpec{
			{
				StorageSource: model.StorageSourceURLDownload,
				URL:           finalURL,
				Path:          mountPath,
			},
		}, err
	}
}

func PartialAdd(numberOfNodes int, store ISetupStorage) ISetupStorage {
	return func(ctx context.Context, driverName model.StorageSourceType, ipfsClients ...*ipfs.Client) ([]model.StorageSpec, error) {
		return store(ctx, driverName, ipfsClients[:numberOfNodes]...)
	}
}

func ManyStores(stores ...ISetupStorage) ISetupStorage {
	return func(ctx context.Context, driverName model.StorageSourceType, ipfsClients ...*ipfs.Client) ([]model.StorageSpec, error) {
		specs := []model.StorageSpec{}
		for _, store := range stores {
			spec, err := store(ctx, driverName, ipfsClients...)
			if err != nil {
				return specs, err
			}
			specs = append(specs, spec...)
		}
		return specs, nil
	}
}

/*

	Results checkers

*/

func FileContains(
	outputFilePath string,
	expectedString string,
	expectedLines int,
) ICheckResults {
	return func(resultsDir string) error {
		outputFile := filepath.Join(resultsDir, outputFilePath)
		resultsContent, err := os.ReadFile(outputFile)
		if err != nil {
			return err
		}

		actualLineCount := len(strings.Split(string(resultsContent), "\n"))
		if actualLineCount != expectedLines {
			return fmt.Errorf("%s: count mismatch:\nExpected: %d\nActual: %d", outputFile, expectedLines, actualLineCount)
		}

		if !strings.Contains(string(resultsContent), expectedString) {
			return fmt.Errorf("%s: content mismatch:\nExpected Contains: %s\nActual: %s", outputFile, expectedString, resultsContent)
		}

		return nil
	}
}

func FileEquals(
	outputFilePath string,
	expectedString string,
) ICheckResults {
	return func(resultsDir string) error {
		outputFile := filepath.Join(resultsDir, outputFilePath)
		resultsContent, err := os.ReadFile(outputFile)
		if err != nil {
			return err
		}

		if string(resultsContent) != expectedString {
			return fmt.Errorf("%s: content mismatch:\nExpected: %s\nActual: %s", outputFile, expectedString, resultsContent)
		}
		return nil
	}
}

func ManyChecks(checks ...ICheckResults) ICheckResults {
	return func(resultsDir string) error {
		for _, check := range checks {
			err := check(resultsDir)
			if err != nil {
				return err
			}
		}
		return nil
	}
}

func WaitUntilComplete(nodes int) []job.CheckStatesFunction {
	return []job.CheckStatesFunction{
		job.WaitThrowErrors([]model.JobStateType{
			model.JobStateError,
		}),
		job.WaitForJobStates(map[model.JobStateType]int{
			model.JobStateCompleted: nodes,
		}),
	}
}
