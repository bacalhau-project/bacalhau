//go:build integration || !unit

// cspell:ignore Dont

package devstack

import (
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy/semantic"
	"github.com/bacalhau-project/bacalhau/pkg/config/configenv"
	"github.com/bacalhau-project/bacalhau/pkg/devstack"
	"github.com/bacalhau-project/bacalhau/pkg/node"

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

// Reuse the docker executor tests but full end to end with libp2p transport
// and 3 nodes
func (suite *DevstackJobSelectionSuite) TestSelectAllJobs() {
	type TestCase struct {
		name            string
		policy          node.JobSelectionPolicy
		nodeCount       int
		addFilesCount   int
		expectedAccepts int
	}

	runTest := func(testCase TestCase) {
		if testCase.nodeCount != testCase.addFilesCount {
			// TODO(forrest) [correctness]: this issue was closed ~2 years ago
			// maybe it's time to run this test now?
			suite.T().Skip("https://github.com/bacalhau-project/bacalhau/issues/361")
		}

		computeConfig, err := node.NewComputeConfigWith(configenv.Testing.Node.ComputeStoragePath, node.ComputeConfigParams{
			JobSelectionPolicy: testCase.policy,
		})
		suite.Require().NoError(err)
		testScenario := scenario.Scenario{
			Stack: &scenario.StackConfig{
				DevStackOptions: &devstack.DevStackOptions{NumberOfHybridNodes: testCase.nodeCount},
				ComputeConfig:   computeConfig,
			},
			Inputs:  scenario.PartialAdd(testCase.addFilesCount, scenario.WasmCsvTransform(suite.T()).Inputs),
			Outputs: scenario.WasmCsvTransform(suite.T()).Outputs,
			Spec:    scenario.WasmCsvTransform(suite.T()).Spec,
			Deal:    model.Deal{Concurrency: testCase.nodeCount},
			JobCheckers: []legacy_job.CheckStatesFunction{
				legacy_job.WaitDontExceedCount(testCase.expectedAccepts),
				legacy_job.WaitExecutionsThrowErrors([]model.ExecutionStateType{
					model.ExecutionStateFailed,
				}),
				legacy_job.WaitForExecutionStates(map[model.ExecutionStateType]int{
					model.ExecutionStateCompleted: testCase.expectedAccepts,
				}),
			},
		}

		suite.RunScenario(testScenario)
	}

	for _, testCase := range []TestCase{

		{
			name:            "all nodes added files, all nodes ran job",
			policy:          node.NewDefaultJobSelectionPolicy(),
			nodeCount:       1,
			addFilesCount:   1,
			expectedAccepts: 1,
		},

		// check we get only 2 when we've only added data to 2
		{
			name:            "only nodes we added data to ran the job",
			policy:          node.NewDefaultJobSelectionPolicy(),
			nodeCount:       3,
			addFilesCount:   2,
			expectedAccepts: 2,
		},

		// check we run on all 3 nodes even though we only added data to 1
		{
			name: "only added files to 1 node but all 3 run it",
			policy: node.JobSelectionPolicy{
				Locality: semantic.Anywhere,
			},
			nodeCount:       3,
			addFilesCount:   1,
			expectedAccepts: 3,
		},
	} {
		suite.Run(testCase.name, func() {
			runTest(testCase)
		})
	}
}
