package scenario

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/filecoin-project/bacalhau/pkg/devstack"
	"github.com/filecoin-project/bacalhau/pkg/ipfs"
	"github.com/filecoin-project/bacalhau/pkg/job"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/node"
	"github.com/filecoin-project/bacalhau/pkg/publicapi"
	"github.com/filecoin-project/bacalhau/pkg/requesternode"
	"github.com/filecoin-project/bacalhau/pkg/system"
	testutils "github.com/filecoin-project/bacalhau/pkg/test/utils"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.opentelemetry.io/otel/trace"
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
	Ctx  context.Context
	Span trace.Span
}

func (s *ScenarioRunner) SetupTest() {
	require.NoError(s.T(), system.InitConfigForTesting(s.T()))

	t := system.GetTracer()
	ctx, rootSpan := system.NewRootSpan(context.Background(), t, s.T().Name())
	s.Ctx = ctx
	s.Span = rootSpan

	s.T().Cleanup(func() { _ = system.CleanupTraceProvider() })
}

func (s *ScenarioRunner) TearDownTest() {
	s.Span.End()
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
		config.DevStackOptions = &devstack.DevStackOptions{NumberOfNodes: 1}
	}

	if config.RequesterNodeConfig == nil {
		conf := requesternode.NewDefaultRequesterNodeConfig()
		config.RequesterNodeConfig = &conf
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
		*config.RequesterNodeConfig,
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
	testutils.MaybeNeedDocker(s.T(), spec.Engine == model.EngineDocker)

	stack, cm := s.setupStack(scenario.Stack)

	// Check that the stack has the appropriate executor installed
	for _, node := range stack.Nodes {
		executor, err := node.Executors.GetExecutor(s.Ctx, spec.Engine)
		require.NoError(s.T(), err)

		isInstalled, err := executor.IsInstalled(s.Ctx)
		require.NoError(s.T(), err)
		require.True(s.T(), isInstalled, "Expected %v to be installed on node %s", spec.Engine, node.HostID)
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

	j.Deal = scenario.Deal
	if j.Deal.Concurrency < 1 {
		j.Deal.Concurrency = 1
	}

	apiClient := publicapi.NewAPIClient(stack.Nodes[0].APIServer.GetURI())
	submittedJob, err := apiClient.Submit(s.Ctx, j, nil)
	require.NoError(s.T(), err)

	// Wait for job to complete
	resolver := apiClient.GetJobStateResolver()
	checkers := scenario.JobCheckers
	shards := job.GetJobTotalExecutionCount(submittedJob)
	err = resolver.Wait(s.Ctx, submittedJob.ID, shards, checkers...)
	require.NoError(s.T(), err)

	// Check outputs
	results, err := apiClient.GetResults(s.Ctx, submittedJob.ID)
	require.NoError(s.T(), err)

	resultsDir = s.T().TempDir()
	swarmAddresses, err := stack.Nodes[0].IPFSClient.SwarmAddresses(s.Ctx)
	require.NoError(s.T(), err)

	err = ipfs.DownloadJob(s.Ctx, cm, spec.Outputs, results, ipfs.IPFSDownloadSettings{
		TimeoutSecs:    5,
		OutputDir:      resultsDir,
		IPFSSwarmAddrs: strings.Join(swarmAddresses, ","),
	})
	require.NoError(s.T(), err)

	if scenario.ResultsChecker != nil {
		err = scenario.ResultsChecker(filepath.Join(resultsDir, ipfs.DownloadVolumesFolderName))
		require.NoError(s.T(), err)
	}

	return resultsDir
}
