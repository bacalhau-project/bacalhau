//go:build integration || !unit

package devstack

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/executor"
	"github.com/bacalhau-project/bacalhau/pkg/executor/noop"
	"github.com/bacalhau-project/bacalhau/pkg/job"
	testutils "github.com/bacalhau-project/bacalhau/pkg/test/utils"

	"github.com/bacalhau-project/bacalhau/pkg/devstack"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/bacalhau-project/bacalhau/pkg/test/scenario"
)

type TargetAllSuite struct {
	scenario.ScenarioRunner
}

func TestTargetAllSuite(t *testing.T) {
	suite.Run(t, new(TargetAllSuite))
}

func (suite *TargetAllSuite) TestCanTargetZeroNodes() {
	testCase := scenario.Scenario{
		Stack: &scenario.StackConfig{DevStackOptions: &devstack.DevStackOptions{
			NumberOfHybridNodes:        0,
			NumberOfRequesterOnlyNodes: 1,
			NumberOfComputeOnlyNodes:   0,
		}},
		Spec:          testutils.MakeSpecWithOpts(suite.T()),
		Deal:          model.Deal{TargetingMode: model.TargetAll},
		SubmitChecker: scenario.SubmitJobSuccess(),
		JobCheckers:   scenario.WaitUntilSuccessful(0),
	}

	suite.RunScenario(testCase)
}

func (suite *TargetAllSuite) TestCanTargetSingleNode() {
	testCase := scenario.Scenario{
		Stack: &scenario.StackConfig{DevStackOptions: &devstack.DevStackOptions{
			NumberOfHybridNodes:        0,
			NumberOfRequesterOnlyNodes: 1,
			NumberOfComputeOnlyNodes:   1,
		}},
		Spec:          testutils.MakeSpecWithOpts(suite.T()),
		Deal:          model.Deal{TargetingMode: model.TargetAll},
		SubmitChecker: scenario.SubmitJobSuccess(),
		JobCheckers: []job.CheckStatesFunction{
			job.WaitForSuccessfulCompletion(),
			job.WaitForExecutionStates(map[model.ExecutionStateType]int{
				model.ExecutionStateCompleted: 1,
			}),
		},
	}

	suite.RunScenario(testCase)
}

func (suite *TargetAllSuite) TestCanTargetMultipleNodes() {
	testCase := scenario.Scenario{
		Stack: &scenario.StackConfig{DevStackOptions: &devstack.DevStackOptions{
			NumberOfHybridNodes:        0,
			NumberOfRequesterOnlyNodes: 1,
			NumberOfComputeOnlyNodes:   5,
		}},
		Spec:          testutils.MakeSpecWithOpts(suite.T()),
		Deal:          model.Deal{TargetingMode: model.TargetAll},
		SubmitChecker: scenario.SubmitJobSuccess(),
		JobCheckers: []job.CheckStatesFunction{
			job.WaitForSuccessfulCompletion(),
			job.WaitForExecutionStates(map[model.ExecutionStateType]int{
				model.ExecutionStateCompleted: 5,
			}),
		},
	}

	suite.RunScenario(testCase)
}

func (suite *TargetAllSuite) TestPartialFailure() {
	var hasFailed atomic.Bool

	testCase := scenario.Scenario{
		Stack: &scenario.StackConfig{
			DevStackOptions: &devstack.DevStackOptions{
				NumberOfHybridNodes:        0,
				NumberOfRequesterOnlyNodes: 1,
				NumberOfComputeOnlyNodes:   2,
			},
			ExecutorConfig: noop.ExecutorConfig{
				ExternalHooks: noop.ExecutorConfigExternalHooks{
					JobHandler: func(ctx context.Context, _ string, resultsDir string) (*model.RunCommandResult, error) {
						if !hasFailed.Swap(true) {
							return executor.FailResult(fmt.Errorf("oh no"))
						} else {
							return executor.WriteJobResults(resultsDir, nil, nil, 0, nil, executor.OutputLimits{
								MaxStdoutFileLength:   system.MaxStdoutFileLength,
								MaxStdoutReturnLength: system.MaxStdoutReturnLength,
								MaxStderrFileLength:   system.MaxStderrFileLength,
								MaxStderrReturnLength: system.MaxStderrReturnLength,
							})
						}
					},
				},
			},
		},
		Spec:          testutils.MakeSpecWithOpts(suite.T()),
		Deal:          model.Deal{TargetingMode: model.TargetAll},
		SubmitChecker: scenario.SubmitJobSuccess(),
		JobCheckers: []job.CheckStatesFunction{
			job.WaitForUnsuccessfulCompletion(),
			job.WaitForExecutionStates(map[model.ExecutionStateType]int{
				model.ExecutionStateCompleted: 1,
				model.ExecutionStateFailed:    1,
			}),
		},
	}

	suite.RunScenario(testCase)
}
