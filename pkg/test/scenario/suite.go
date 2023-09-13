package scenario

import (
	"context"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/lib/provider"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/client"
	"github.com/bacalhau-project/bacalhau/pkg/setup"

	"github.com/bacalhau-project/bacalhau/pkg/devstack"
	"github.com/bacalhau-project/bacalhau/pkg/docker"
	"github.com/bacalhau-project/bacalhau/pkg/downloader"
	"github.com/bacalhau-project/bacalhau/pkg/downloader/ipfs"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/node"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/bacalhau-project/bacalhau/pkg/telemetry"
	testutils "github.com/bacalhau-project/bacalhau/pkg/test/teststack"
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
	logger.ConfigureTestLogging(s.T())
	fsRepo := setup.SetupBacalhauRepoForTesting(s.T())
	repoPath, err := fsRepo.Path()
	if err != nil {
		s.T().Fatal(err)
	}
	s.T().Setenv("BACALHAU_DIR", repoPath)

	s.Ctx = context.Background()

	s.T().Cleanup(func() { _ = telemetry.Cleanup() })
}

func (s *ScenarioRunner) prepareStorage(stack *devstack.DevStack, getStorage SetupStorage) []model.StorageSpec {
	if getStorage == nil {
		return []model.StorageSpec{}
	}

	clients := stack.IPFSClients()
	s.Require().GreaterOrEqual(len(clients), 1, "No IPFS clients to upload to?")

	storageList, stErr := getStorage(s.Ctx, model.StorageSourceIPFS, stack.IPFSClients()...)
	s.Require().NoError(stErr)

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

	if config.RequesterConfig.JobDefaults.ExecutionTimeout == 0 {
		config.RequesterConfig = node.NewRequesterConfigWithDefaults()
	}

	if config.ComputeConfig.TotalResourceLimits.IsZero() {
		// TODO(forrest): [correctness] if the provided compute config has one `0` field we override the whole thing.
		// we probably want to merge these instead.
		cfg, err := node.NewComputeConfigWithDefaults()
		s.Require().NoError(err)
		config.ComputeConfig = cfg
	}
	stack := testutils.Setup(s.Ctx, s.T(),
		append(config.DevStackOptions.Options(),
			devstack.WithComputeConfig(config.ComputeConfig),
			devstack.WithRequesterConfig(config.RequesterConfig),
			testutils.WithNoopExecutor(config.ExecutorConfig),
		)...,
	)

	return stack, stack.Nodes[0].CleanupManager
}

// RunScenario runs the Scenario.
//
// Spin up a devstack, execute the job, check the results, and tear down the
// devstack.
func (s *ScenarioRunner) RunScenario(scenario Scenario) (resultsDir string) {
	spec := scenario.Spec
	docker.EngineSpecRequiresDocker(s.T(), spec.EngineSpec)

	stack, cm := s.setupStack(scenario.Stack)

	s.T().Log("Setting up storage")
	spec.Inputs = s.prepareStorage(stack, scenario.Inputs)
	spec.Outputs = scenario.Outputs
	if spec.Outputs == nil {
		spec.Outputs = []model.StorageSpec{}
	}

	s.T().Log("Submitting job")
	j, err := model.NewJobWithSaneProductionDefaults()
	s.Require().NoError(err)

	j.Spec = spec
	s.Require().True(model.IsValidEngine(j.Spec.EngineSpec.Engine()))
	if !model.IsValidPublisher(j.Spec.PublisherSpec.Type) {
		j.Spec.PublisherSpec = model.PublisherSpec{
			Type: model.PublisherIpfs,
		}
	}

	j.Spec.Deal = scenario.Deal
	if j.Spec.Deal.Concurrency < 1 {
		j.Spec.Deal.Concurrency = 1
	}

	apiServer := stack.Nodes[0].APIServer
	apiClient := client.NewAPIClient(apiServer.Address, apiServer.Port)
	submittedJob, submitError := apiClient.Submit(s.Ctx, j)
	if scenario.SubmitChecker == nil {
		scenario.SubmitChecker = SubmitJobSuccess()
	}
	err = scenario.SubmitChecker(submittedJob, submitError)
	s.Require().NoError(err)

	// exit if the test expects submission to fail as no further assertions can be made
	if submitError != nil {
		return
	}

	s.T().Log("Waiting for job")
	resolver := apiClient.GetJobStateResolver()
	err = resolver.Wait(s.Ctx, submittedJob.Metadata.ID, scenario.JobCheckers...)
	s.Require().NoError(err)

	// Check outputs
	if scenario.ResultsChecker != nil {
		s.T().Log("Checking output")
		results, err := apiClient.GetResults(s.Ctx, submittedJob.Metadata.ID)
		s.Require().NoError(err)

		resultsDir = s.T().TempDir()
		var swarmAddresses []string
		for _, n := range stack.Nodes {
			addrs, err := n.IPFSClient.SwarmAddresses(s.Ctx)
			s.Require().NoError(err)
			swarmAddresses = append(swarmAddresses, addrs...)
		}

		viper.Set(types.NodeIPFSSwarmAddresses, swarmAddresses)
		viper.Set(types.NodeIPFSPrivateInternal, true)

		downloaderSettings := &model.DownloaderSettings{
			Timeout:   time.Second * 10,
			OutputDir: resultsDir,
		}

		ipfsDownloader := ipfs.NewIPFSDownloader(cm, downloaderSettings)
		s.Require().NoError(err)

		downloaderProvider := provider.NewMappedProvider(map[string]downloader.Downloader{
			models.StorageSourceIPFS: ipfsDownloader,
		})

		err = downloader.DownloadResults(s.Ctx, results, downloaderProvider, downloaderSettings)
		s.Require().NoError(err)

		err = scenario.ResultsChecker(resultsDir)
		s.Require().NoError(err)
	}

	return resultsDir
}
