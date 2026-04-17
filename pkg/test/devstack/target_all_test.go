//go:build integration || !unit

package devstack

import (
	"context"
	"fmt"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/models"

	"github.com/bacalhau-project/bacalhau/pkg/devstack"
	"github.com/bacalhau-project/bacalhau/pkg/executor"
	"github.com/bacalhau-project/bacalhau/pkg/executor/noop"
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
		Stack: &scenario.StackConfig{DevStackOptions: []devstack.ConfigOption{
			devstack.WithNumberOfRequesterOnlyNodes(1),
		}},
		Job: &models.Job{
			Name: suite.T().Name(),
			Type: models.JobTypeOps,
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
		SubmitChecker: scenario.SubmitJobSuccess(),
		JobCheckers:   scenario.WaitUntilSuccessful(0),
	}

	suite.RunScenario(testCase)
}

func (suite *TargetAllSuite) TestCanTargetSingleNode() {
	testCase := scenario.Scenario{
		Stack: &scenario.StackConfig{
			DevStackOptions: []devstack.ConfigOption{
				devstack.WithNumberOfHybridNodes(1),
			},
		},
		Job: &models.Job{
			Name: suite.T().Name(),
			Type: models.JobTypeOps,
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
		SubmitChecker: scenario.SubmitJobSuccess(),
		JobCheckers: []scenario.StateChecks{
			scenario.WaitForSuccessfulCompletion(),
			scenario.WaitForExecutionStates(map[models.ExecutionStateType]int{
				models.ExecutionStateCompleted: 1,
			}),
		},
	}

	suite.RunScenario(testCase)
}

func (suite *TargetAllSuite) TestCanTargetMultipleNodes() {
	testCase := scenario.Scenario{
		Stack: &scenario.StackConfig{DevStackOptions: []devstack.ConfigOption{
			devstack.WithNumberOfHybridNodes(1),
			devstack.WithNumberOfComputeOnlyNodes(4),
		}},
		Job: &models.Job{
			Name: suite.T().Name(),
			Type: models.JobTypeOps,
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
		SubmitChecker: scenario.SubmitJobSuccess(),
		JobCheckers: []scenario.StateChecks{
			scenario.WaitForSuccessfulCompletion(),
			scenario.WaitForExecutionStates(map[models.ExecutionStateType]int{
				models.ExecutionStateCompleted: 5,
			}),
		},
	}

	suite.RunScenario(testCase)
}

func (suite *TargetAllSuite) TestPartialFailure() {
	var hasFailed atomic.Bool

	testCase := scenario.Scenario{
		Stack: &scenario.StackConfig{
			DevStackOptions: []devstack.ConfigOption{
				devstack.WithNumberOfHybridNodes(1),
				devstack.WithNumberOfComputeOnlyNodes(1),
			},
			ExecutorConfig: noop.ExecutorConfig{
				ExternalHooks: noop.ExecutorConfigExternalHooks{
					JobHandler: func(ctx context.Context, execContext noop.ExecutionContext) (*models.RunCommandResult, error) {
						if !hasFailed.Swap(true) {
							return executor.FailResult(fmt.Errorf("oh no"))
						} else {
							return executor.WriteJobResults(execContext.ExecutionDir, nil, nil, 0, nil, executor.OutputLimits{
								MaxStdoutFileLength:   system.MaxStdoutFileLength,
								MaxStdoutReturnLength: system.MaxStdoutReturnLength,
								MaxStderrFileLength:   system.MaxStderrFileLength,
								MaxStderrReturnLength: system.MaxStderrReturnLength,
							}), nil
						}
					},
				},
			},
		},
		Job: &models.Job{
			Name: suite.T().Name(),
			Type: models.JobTypeOps,
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
		SubmitChecker: scenario.SubmitJobSuccess(),
		JobCheckers: []scenario.StateChecks{
			scenario.WaitForUnsuccessfulCompletion(),
			scenario.WaitForExecutionStates(map[models.ExecutionStateType]int{
				models.ExecutionStateCompleted: 1,
				models.ExecutionStateFailed:    1,
			}),
		},
	}

	suite.RunScenario(testCase)
}
