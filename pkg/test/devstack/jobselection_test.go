//go:build integration

package devstack

import (
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/devstack"
	"github.com/filecoin-project/bacalhau/pkg/node"

	"github.com/filecoin-project/bacalhau/pkg/job"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/test/scenario"
	"github.com/stretchr/testify/suite"
)

type DevstackJobSelectionSuite struct {
	scenario.ScenarioRunner
}

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestDevstackJobSelectionSuite(t *testing.T) {
	suite.Run(t, new(DevstackJobSelectionSuite))
}

// Re-use the docker executor tests but full end to end with libp2p transport
// and 3 nodes
func (suite *DevstackJobSelectionSuite) TestSelectAllJobs() {
	type TestCase struct {
		name            string
		policy          model.JobSelectionPolicy
		nodeCount       int
		addFilesCount   int
		expectedAccepts int
	}

	runTest := func(testCase TestCase) {
		if testCase.nodeCount != testCase.addFilesCount {
			suite.T().Skip("https://github.com/filecoin-project/bacalhau/issues/361")
		}

		testScenario := scenario.Scenario{
			Stack: &scenario.StackConfig{
				DevStackOptions: &devstack.DevStackOptions{NumberOfHybridNodes: testCase.nodeCount},
				ComputeConfig: node.NewComputeConfigWith(node.ComputeConfigParams{
					JobSelectionPolicy: testCase.policy,
				}),
			},
			Inputs:   scenario.PartialAdd(testCase.addFilesCount, scenario.WasmCsvTransform.Inputs),
			Contexts: scenario.WasmCsvTransform.Contexts,
			Outputs:  scenario.WasmCsvTransform.Outputs,
			Spec:     scenario.WasmCsvTransform.Spec,
			Deal:     model.Deal{Concurrency: testCase.nodeCount},
			JobCheckers: []job.CheckStatesFunction{
				job.WaitDontExceedCount(testCase.expectedAccepts),
				job.WaitExecutionsThrowErrors([]model.ExecutionStateType{
					model.ExecutionStateFailed,
				}),
				job.WaitForExecutionStates(map[model.ExecutionStateType]int{
					model.ExecutionStateCompleted: testCase.expectedAccepts,
				}),
			},
		}

		suite.RunScenario(testScenario)
	}

	for _, testCase := range []TestCase{

		{
			name:            "all nodes added files, all nodes ran job",
			policy:          model.NewDefaultJobSelectionPolicy(),
			nodeCount:       3,
			addFilesCount:   3,
			expectedAccepts: 3,
		},

		// check we get only 2 when we've only added data to 2
		{
			name:            "only nodes we added data to ran the job",
			policy:          model.NewDefaultJobSelectionPolicy(),
			nodeCount:       3,
			addFilesCount:   2,
			expectedAccepts: 2,
		},

		// check we run on all 3 nodes even though we only added data to 1
		{
			name: "only added files to 1 node but all 3 run it",
			policy: model.JobSelectionPolicy{
				Locality: model.Anywhere,
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
