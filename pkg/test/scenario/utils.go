package scenario

import (
	"context"
	"fmt"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/filecoin-project/bacalhau/pkg/computenode"
	"github.com/filecoin-project/bacalhau/pkg/ipfs"
	"github.com/filecoin-project/bacalhau/pkg/job"
	"github.com/filecoin-project/bacalhau/pkg/requesternode"

	"github.com/filecoin-project/bacalhau/pkg/devstack"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	apicopy "github.com/filecoin-project/bacalhau/pkg/storage/ipfs_apicopy"
	fusedocker "github.com/filecoin-project/bacalhau/pkg/storage/ipfs_fusedocker"
	"github.com/rs/zerolog/log"
)

// A Scenario represents a repeatable test case of submitting a job against a
// Bacalhau network.
//
// The Scenario defines:
//
// * the topology and configuration of network that is required
// * the job that will be submitted
// * the conditions for the job to be considered successful or not
type Scenario struct {
	Name string

	// An optional set of configuration options that define the network of nodes
	// that the job will be run against
	Stack *StackConfig

	// Setup routines which define data available to the job, potentially sharded
	Inputs ISetupStorage

	// Setup routines which define data available to the job, for every shard
	Contexts ISetupStorage

	// Output volumes that must be available to the job
	Outputs []model.StorageSpec

	// The job specification
	Spec model.Spec

	// The job deal
	Deal model.Deal

	// A function that will decide whether or not the job was successful
	ResultsChecker ICheckResults

	// A set of checkers that will decide when the job has completed, and maybe
	// whether it was successful or not
	JobCheckers []job.CheckStatesFunction
}

// All the information that is needed to uniquely define a devstack.
type StackConfig struct {
	*devstack.DevStackOptions
	*computenode.ComputeNodeConfig
	*requesternode.RequesterNodeConfig
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
	return func(_ context.Context, _ model.StorageSourceType, _ ...*ipfs.Client) ([]model.StorageSpec, error) {
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
