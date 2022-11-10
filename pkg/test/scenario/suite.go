package scenario

import (
	"context"
	"path/filepath"
	"strings"

	"github.com/filecoin-project/bacalhau/pkg/computenode"
	"github.com/filecoin-project/bacalhau/pkg/devstack"
	"github.com/filecoin-project/bacalhau/pkg/ipfs"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/publicapi"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.opentelemetry.io/otel/trace"
)

type ScenarioTestSuite interface {
	suite.SetupTestSuite
	suite.TearDownTestSuite
	suite.TestingSuite
}

type ScenarioRunner struct {
	suite.Suite
	Cm    *system.CleanupManager
	Ctx   context.Context
	Span  trace.Span
	Stack *devstack.DevStack
}

func (s *ScenarioRunner) SetupTest() {
	require.NoError(s.T(), system.InitConfigForTesting())
	s.Cm = system.NewCleanupManager()
	s.T().Cleanup(s.Cm.Cleanup)

	t := system.GetTracer()
	ctx, rootSpan := system.NewRootSpan(context.Background(), t, s.T().Name())
	s.Ctx = ctx
	s.Span = rootSpan

	s.Cm.RegisterCallback(system.CleanupTraceProvider)
}

func (s *ScenarioRunner) TearDownTest() {
	s.Stack = nil
	s.Span.End()
}

func (s *ScenarioRunner) prepareStorage(getStorage ISetupStorage) []model.StorageSpec {
	if getStorage == nil {
		return []model.StorageSpec{}
	}

	clients := s.Stack.IPFSClients()
	require.GreaterOrEqual(s.T(), len(clients), 1, "No IPFS clients to upload to?")

	storageList, stErr := getStorage(s.Ctx, model.StorageSourceIPFS, s.Stack.IPFSClients()[:1]...)
	require.NoError(s.T(), stErr)

	return storageList
}

func (s *ScenarioRunner) SetupStack(opts *devstack.DevStackOptions, cnconf computenode.ComputeNodeConfig) *devstack.DevStack {
	stack, err := devstack.NewStandardDevStack(
		s.Ctx,
		s.Cm,
		*opts,
		cnconf,
	)
	require.NoError(s.T(), err)

	s.Stack = stack
	return s.Stack
}

func (s *ScenarioRunner) RunScenario(scenario TestCase) (resultsDir string) {
	spec := scenario.Spec

	if s.Stack == nil {
		s.SetupStack(&devstack.DevStackOptions{
			NumberOfNodes:     1,
			NumberOfBadActors: 0,
		}, computenode.NewDefaultComputeNodeConfig())
	}

	// Check that the stack has the appropriate executor installed
	for _, node := range s.Stack.Nodes {
		executor, err := node.Executors.GetExecutor(s.Ctx, spec.Engine)
		require.NoError(s.T(), err)

		isInstalled, err := executor.IsInstalled(s.Ctx)
		require.NoError(s.T(), err)
		require.True(s.T(), isInstalled, "Expected %v to be installed on node %s", spec.Engine, node.HostID)
	}

	// TODO: assert network connectivity

	// Setup storage
	spec.Inputs = s.prepareStorage(scenario.Inputs)
	spec.Contexts = s.prepareStorage(scenario.Contexts)
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

	apiClient := publicapi.NewAPIClient(s.Stack.Nodes[0].APIServer.GetURI())
	submittedJob, err := apiClient.Submit(s.Ctx, j, nil)
	require.NoError(s.T(), err)

	// Wait for job to complete
	resolver := apiClient.GetJobStateResolver()
	checkers := scenario.JobCheckers
	err = resolver.Wait(s.Ctx, submittedJob.ID, 1, checkers...) // TODO shards
	require.NoError(s.T(), err)

	// Check outputs
	results, err := apiClient.GetResults(s.Ctx, submittedJob.ID)
	require.NoError(s.T(), err)

	resultsDir = s.T().TempDir()
	swarmAddresses, err := s.Stack.Nodes[0].IPFSClient.SwarmAddresses(s.Ctx)
	require.NoError(s.T(), err)

	err = ipfs.DownloadJob(s.Ctx, s.Cm, spec.Outputs, results, ipfs.IPFSDownloadSettings{
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
