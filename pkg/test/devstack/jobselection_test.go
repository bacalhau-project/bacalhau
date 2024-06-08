//go:build integration || !unit

// cspell:ignore Dont

package devstack

import (
	"context"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy/semantic"
	"github.com/bacalhau-project/bacalhau/pkg/config/configenv"
	"github.com/bacalhau-project/bacalhau/pkg/devstack"
	"github.com/bacalhau-project/bacalhau/pkg/node"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator/retry"
	testutils "github.com/bacalhau-project/bacalhau/pkg/test/utils"

	"github.com/stretchr/testify/suite"

	legacy_job "github.com/bacalhau-project/bacalhau/pkg/legacyjob"
	_ "github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/model"
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
	type TestCase struct {
		name      string
		policy    node.JobSelectionPolicy
		addFiles  bool
		completed int
		rejected  int
		failed    int
	}

	runTest := func(testCase TestCase) {
		computeConfig, err := node.NewComputeConfigWith(configenv.Testing.Node.ComputeStoragePath, node.ComputeConfigParams{
			JobSelectionPolicy: testCase.policy,
		})
		suite.Require().NoError(err)

		requesterConfig, err := node.NewRequesterConfigWith(node.RequesterConfigParams{
			RetryStrategy: retry.NewFixedStrategy(retry.FixedStrategyParams{ShouldRetry: false}),
		})
		suite.Require().NoError(err)

		rootSourceDir := suite.T().TempDir()

		var inputs scenario.SetupStorage
		if testCase.addFiles {
			inputs = scenario.StoredText(rootSourceDir, "job selection", "/inputs")
		} else {
			inputs = func(ctx context.Context) ([]model.StorageSpec, error) {
				sourceFile, err := scenario.CreateSourcePath(rootSourceDir)
				if err != nil {
					return nil, err
				}
				return []model.StorageSpec{
					{
						StorageSource: model.StorageSourceLocalDirectory,
						SourcePath:    sourceFile,
						Path:          "/inputs",
					},
				}, nil
			}
		}

		testScenario := scenario.Scenario{
			Stack: &scenario.StackConfig{
				DevStackOptions: &devstack.DevStackOptions{
					AllowListedLocalPaths: []string{rootSourceDir + "/*"},
				},
				ComputeConfig:   computeConfig,
				RequesterConfig: requesterConfig,
			},
			Inputs: inputs,
			Spec:   testutils.MakeNoopJob(suite.T()).Spec,
			JobCheckers: []legacy_job.CheckStatesFunction{
				legacy_job.WaitForExecutionStates(map[model.ExecutionStateType]int{
					model.ExecutionStateCompleted:         testCase.completed,
					model.ExecutionStateAskForBidRejected: testCase.rejected,
					model.ExecutionStateFailed:            testCase.failed,
				}),
			},
		}

		suite.RunScenario(testScenario)
	}

	for _, testCase := range []TestCase{
		{
			name: "Local: Add files, Accept job",
			policy: node.JobSelectionPolicy{
				Locality: semantic.Local,
			},
			addFiles:  true,
			completed: 1,
		},

		{
			name: "Local: Don't add files, Reject job",
			policy: node.JobSelectionPolicy{
				Locality: semantic.Local,
			},
			addFiles: false,
			rejected: 1,
		},
		{
			name: "Anywhere: Don't add files, Fail job",
			policy: node.JobSelectionPolicy{
				Locality: semantic.Anywhere,
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
