package scenario

import (
	"context"
	"path/filepath"
	"strings"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/downloader/ipfs"
	"github.com/filecoin-project/bacalhau/pkg/telemetry"

	"github.com/filecoin-project/bacalhau/pkg/downloader"

	"github.com/filecoin-project/bacalhau/pkg/devstack"
	"github.com/filecoin-project/bacalhau/pkg/docker"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/node"
	"github.com/filecoin-project/bacalhau/pkg/requester/publicapi"
	"github.com/filecoin-project/bacalhau/pkg/system"
	testutils "github.com/filecoin-project/bacalhau/pkg/test/utils"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type ScenarioTestSuite interface {
	suite.SetupTestSuite
	suite.TearDownTestSuite
	suite.TestingSuite
}

// The ScenarioRunner is an object that can run a Scenario.
//
// It will spin up an appropriate Devstack for the Scenario, submit and wait for
// the job to complete, and then make assertions against the results of the job.
//
// ScenarioRunner implements a number of testify/suite interfaces making it
// appropriate as the basis for a test suite. If a test suite composes itself
// from the ScenarioRunner then default set up and tear down methods that
// instrument and configure the test will be used. Test suites should not define
// their own set up or tear down routines.
type ScenarioRunner struct {
	suite.Suite
	Ctx context.Context
}

func (s *ScenarioRunner) SetupTest() {
	require.NoError(s.T(), system.InitConfigForTesting(s.T()))

	s.Ctx = context.Background()

	s.T().Cleanup(func() { _ = telemetry.Cleanup() })
}

func (s *ScenarioRunner) prepareStorage(stack *devstack.DevStack, getStorage SetupStorage) []model.StorageSpec {
	if getStorage == nil {
		return []model.StorageSpec{}
	}

	clients := stack.IPFSClients()
	require.GreaterOrEqual(s.T(), len(clients), 1, "No IPFS clients to upload to?")

	storageList, stErr := getStorage(s.Ctx, model.StorageSourceIPFS, stack.IPFSClients()...)
	require.NoError(s.T(), stErr)

	return storageList
}

// Set up the test devstack according to the passed options. By default, the
// devstack will have 1 node with local only data and no timeouts.
func (s *ScenarioRunner) setupStack(config *StackConfig) (*devstack.DevStack, *system.CleanupManager) {
	if config == nil {
		config = &StackConfig{}
	}

	if config.DevStackOptions == nil {
		config.DevStackOptions = &devstack.DevStackOptions{NumberOfHybridNodes: 1}
	}

	if config.RequesterConfig.DefaultJobExecutionTimeout == 0 {
		config.RequesterConfig = node.NewRequesterConfigWithDefaults()
	}

	empty := model.ResourceUsageData{}
	if config.ComputeConfig.TotalResourceLimits == empty {
		config.ComputeConfig = node.NewComputeConfigWithDefaults()
	}

	stack := testutils.SetupTestWithNoopExecutor(
		s.Ctx,
		s.T(),
		*config.DevStackOptions,
		config.ComputeConfig,
		config.RequesterConfig,
		config.ExecutorConfig,
	)

	return stack, stack.Nodes[0].CleanupManager
}

// RunScenario runs the Scenario.
//
// Spin up a devstack, execute the job, check the results, and tear down the
// devstack.
func (s *ScenarioRunner) RunScenario(scenario Scenario) (resultsDir string) {
	spec := scenario.Spec
	docker.MaybeNeedDocker(s.T(), spec.Engine == model.EngineDocker)

	stack, cm := s.setupStack(scenario.Stack)

	// Check that the stack has the appropriate executor installed
	for _, node := range stack.Nodes {
		executor, err := node.ComputeNode.Executors.Get(s.Ctx, spec.Engine)
		require.NoError(s.T(), err)

		isInstalled, err := executor.IsInstalled(s.Ctx)
		require.NoError(s.T(), err)
		require.True(s.T(), isInstalled, "Expected %v to be installed on node %s", spec.Engine, node.Host.ID().String())
	}

	// TODO: assert network connectivity

	// Setup storage
	spec.Inputs = s.prepareStorage(stack, scenario.Inputs)
	spec.Contexts = s.prepareStorage(stack, scenario.Contexts)
	spec.Outputs = scenario.Outputs
	if spec.Outputs == nil {
		spec.Outputs = []model.StorageSpec{}
	}

	// Setup job and submit
	j, err := model.NewJobWithSaneProductionDefaults()
	require.NoError(s.T(), err)

	j.Spec = spec
	require.True(s.T(), model.IsValidEngine(j.Spec.Engine))
	if !model.IsValidVerifier(j.Spec.Verifier) {
		j.Spec.Verifier = model.VerifierNoop
	}
	if !model.IsValidPublisher(j.Spec.Publisher) {
		j.Spec.Publisher = model.PublisherIpfs
	}

	j.Spec.Deal = scenario.Deal
	if j.Spec.Deal.Concurrency < 1 {
		j.Spec.Deal.Concurrency = 1
	}

	apiClient := publicapi.NewRequesterAPIClient(stack.Nodes[0].APIServer.GetURI())
	submittedJob, submitError := apiClient.Submit(s.Ctx, j)
	if scenario.SubmitChecker == nil {
		scenario.SubmitChecker = SubmitJobSuccess()
	}
	err = scenario.SubmitChecker(submittedJob, submitError)
	require.NoError(s.T(), err)

	// exit if the test expects submission to fail as no further assertions can be made
	if submitError != nil {
		return
	}
	// Wait for job to complete
	resolver := apiClient.GetJobStateResolver()
	checkers := scenario.JobCheckers
	err = resolver.Wait(s.Ctx, submittedJob.Metadata.ID, checkers...)
	require.NoError(s.T(), err)

	// Check outputs
	results, err := apiClient.GetResults(s.Ctx, submittedJob.Metadata.ID)
	require.NoError(s.T(), err)

	resultsDir = s.T().TempDir()
	swarmAddresses, err := stack.Nodes[0].IPFSClient.SwarmAddresses(s.Ctx)
	require.NoError(s.T(), err)

	downloaderSettings := &model.DownloaderSettings{
		Timeout:        time.Second * 5,
		OutputDir:      resultsDir,
		IPFSSwarmAddrs: strings.Join(swarmAddresses, ","),
	}

	ipfsDownloader := ipfs.NewIPFSDownloader(cm, downloaderSettings)
	require.NoError(s.T(), err)

	downloaderProvider := model.NewMappedProvider(map[model.StorageSourceType]downloader.Downloader{
		model.StorageSourceIPFS: ipfsDownloader,
	})

	err = downloader.DownloadJob(s.Ctx, spec.Outputs, results, downloaderProvider, downloaderSettings)
	require.NoError(s.T(), err)

	if scenario.ResultsChecker != nil {
		err = scenario.ResultsChecker(filepath.Join(resultsDir, model.DownloadVolumesFolderName))
		require.NoError(s.T(), err)
	}

	return resultsDir
}
