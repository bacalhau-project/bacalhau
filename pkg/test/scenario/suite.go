package scenario

import (
	"context"
	"fmt"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/devstack"
	"github.com/bacalhau-project/bacalhau/pkg/docker"
	"github.com/bacalhau-project/bacalhau/pkg/downloader"
	"github.com/bacalhau-project/bacalhau/pkg/downloader/http"
	"github.com/bacalhau-project/bacalhau/pkg/lib/provider"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	clientv2 "github.com/bacalhau-project/bacalhau/pkg/publicapi/client/v2"
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
	Ctx    context.Context
	Config types.Bacalhau
}

func (s *ScenarioRunner) SetupTest() {
	logger.ConfigureTestLogging(s.T())
	var err error
	s.Config, err = config.NewTestConfig()
	s.Require().NoError(err)

	s.Ctx = context.Background()

	s.T().Cleanup(func() { _ = telemetry.Cleanup() })
}

func (s *ScenarioRunner) prepareStorage(stack *devstack.DevStack, getStorage SetupStorage) []*models.InputSource {
	if getStorage == nil {
		return nil
	}

	storageList, stErr := getStorage(s.Ctx)
	s.Require().NoError(stErr)

	return storageList
}

// Set up the test devstack according to the passed options. By default, the
// devstack will have 1 node with local only data and no timeouts.
func (s *ScenarioRunner) setupStack(stackConfig *StackConfig) (*devstack.DevStack, *system.CleanupManager) {
	if stackConfig == nil {
		stackConfig = &StackConfig{}
	}
	// The order of applying options here matters
	// Start with applying the options applied at the scenario running level
	options := []devstack.ConfigOption{
		devstack.WithBacalhauConfigOverride(s.Config),
		testutils.WithNoopExecutor(stackConfig.ExecutorConfig, s.Config.Engines),
	}
	// Then apply the options passed in the scenario's stack config to allow tests to override the behaviour
	options = append(options, stackConfig.DevStackOptions...)

	// Finally, add a fallback option to ensure that at least one node is created
	options = append(options, devstack.WithAtLeastOneNode())

	stack := testutils.Setup(s.Ctx, s.T(), options...)
	return stack, stack.Nodes[0].CleanupManager
}

// RunScenario runs the Scenario.
//
// Spin up a devstack, execute the job, check the results, and tear down the
// devstack.
func (s *ScenarioRunner) RunScenario(scenario Scenario) string {
	var resultsDir string

	scenario.Job.Normalize()
	job := scenario.Job
	task := job.Task()
	docker.EngineSpecRequiresDocker(s.T(), task.Engine)

	stack, _ := s.setupStack(scenario.Stack)

	s.T().Log("Setting up storage")
	task.InputSources = s.prepareStorage(stack, scenario.Inputs)
	task.ResultPaths = scenario.Outputs

	apiServer := stack.Nodes[0].APIServer
	apiProtocol := "http"
	apiHost := apiServer.Address
	apiPort := apiServer.Port
	api := clientv2.New(fmt.Sprintf("%s://%s:%d", apiProtocol, apiHost, apiPort))

	if scenario.SubmitChecker == nil {
		scenario.SubmitChecker = SubmitJobSuccess()
	}

	s.T().Logf("Submitting job: %v", job)
	putResp, err := api.Jobs().Put(s.Ctx, &apimodels.PutJobRequest{
		Job: job,
	})
	s.Require().NoError(scenario.SubmitChecker(putResp, err))
	// exit if the test expects submission to fail as no further assertions can be made
	if err != nil {
		return resultsDir
	}

	getResp, err := api.Jobs().Get(s.Ctx, &apimodels.GetJobRequest{
		JobID: putResp.JobID,
	})
	s.Require().NoError(err)
	jobID := getResp.Job.ID

	s.T().Log("Waiting for job")
	s.Require().NoError(NewStateResolverFromAPI(api).Wait(s.Ctx, jobID, scenario.JobCheckers...))

	// Check outputs
	if scenario.ResultsChecker != nil {
		s.T().Log("Checking output")
		results, err := api.Jobs().Results(s.Ctx, &apimodels.ListJobResultsRequest{
			JobID: jobID,
		})
		s.Require().NoError(err)

		resultsDir = s.T().TempDir()

		downloaderSettings := &downloader.DownloaderSettings{
			Timeout:   time.Second * 10,
			OutputDir: resultsDir,
		}

		downloaderProvider := provider.NewMappedProvider(map[string]downloader.Downloader{
			models.StorageSourceURL: http.NewHTTPDownloader(),
		})

		err = downloader.DownloadResults(s.Ctx, results.Items, downloaderProvider, downloaderSettings)
		s.Require().NoError(err)

		err = scenario.ResultsChecker(resultsDir)
		s.Require().NoError(err)
	}

	return resultsDir
}
