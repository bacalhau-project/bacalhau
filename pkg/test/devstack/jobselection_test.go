//go:build integration || !unit

// cspell:ignore Dont

package devstack

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/devstack"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/node"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator/retry"
	storage_local "github.com/bacalhau-project/bacalhau/pkg/storage/local_directory"

	_ "github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/test/scenario"
)

type DevstackJobSelectionSuite struct {
	scenario.ScenarioRunner
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestDevstackJobSelectionSuite(t *testing.T) {
	suite.Run(t, new(DevstackJobSelectionSuite))
}

// Reuse the docker executor tests but full end to end with transport layer and 3 nodes
func (suite *DevstackJobSelectionSuite) TestSelectAllJobs() {
	suite.T().Skip("Test makes assertions on data locality, a feature no longer supported.")
	type TestCase struct {
		name      string
		policy    types.JobAdmissionControl
		addFiles  bool
		completed int
		rejected  int
		failed    int
	}

	runTest := func(testCase TestCase) {
		rootSourceDir := suite.T().TempDir()

		var inputs scenario.SetupStorage
		if testCase.addFiles {
			inputs = scenario.StoredText(rootSourceDir, "job selection", "/inputs")
		} else {
			inputs = func(ctx context.Context) ([]*models.InputSource, error) {
				sourceFile, err := scenario.CreateSourcePath(rootSourceDir)
				if err != nil {
					return nil, err
				}
				localSource, err := storage_local.NewSpecConfig(sourceFile, false)
				if err != nil {
					return nil, err
				}
				return []*models.InputSource{
					{
						Target: "/inputs",
						Source: localSource,
					},
				}, nil
			}
		}

		testScenario := scenario.Scenario{
			Stack: &scenario.StackConfig{
				DevStackOptions: []devstack.ConfigOption{
					devstack.WithAllowListedLocalPaths([]string{rootSourceDir + scenario.AllowedListedLocalPathsSuffix}),
					devstack.WithBacalhauConfigOverride(types.Bacalhau{
						JobAdmissionControl: testCase.policy,
					}),
					devstack.WithSystemConfig(node.SystemConfig{
						RetryStrategy: retry.NewFixedStrategy(retry.FixedStrategyParams{ShouldRetry: false}),
					}),
				},
			},
			Inputs: inputs,
			Job: &models.Job{
				Name:  suite.T().Name(),
				Type:  models.JobTypeBatch,
				Count: 1,
				Tasks: []*models.Task{
					{
						Name: suite.T().Name(),
						Engine: &models.SpecConfig{
							Type:   models.EngineNoop,
							Params: make(map[string]interface{}),
						},
					},
				},
			},
			JobCheckers: []scenario.StateChecks{
				scenario.WaitForExecutionStates(map[models.ExecutionStateType]int{
					models.ExecutionStateCompleted:         testCase.completed,
					models.ExecutionStateAskForBidRejected: testCase.rejected,
					models.ExecutionStateFailed:            testCase.failed,
				}),
			},
		}

		suite.RunScenario(testScenario)
	}

	for _, testCase := range []TestCase{
		{
			name: "Local: Add files, Accept job",
			policy: types.JobAdmissionControl{
				Locality: models.Local,
			},
			addFiles:  true,
			completed: 1,
		},

		{
			name: "Local: Don't add files, Reject job",
			policy: types.JobAdmissionControl{
				Locality: models.Local,
			},
			addFiles: false,
			rejected: 1,
		},
		{
			name: "Anywhere: Don't add files, Fail job",
			policy: types.JobAdmissionControl{
				Locality: models.Anywhere,
			},
			addFiles: false,
			failed:   1,
		},
	} {
		suite.Run(testCase.name, func() {
			runTest(testCase)
		})
	}
}
