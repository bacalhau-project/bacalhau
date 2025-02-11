//go:build integration || !unit

package devstack

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/devstack"
	"github.com/bacalhau-project/bacalhau/pkg/executor"
	"github.com/bacalhau-project/bacalhau/pkg/executor/noop"
	_ "github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/bacalhau-project/bacalhau/pkg/test/scenario"
)

type DevstackTimeoutSuite struct {
	scenario.ScenarioRunner
}

func TestDevstackTimeoutSuite(t *testing.T) {
	suite.Run(t, new(DevstackTimeoutSuite))
}

func (suite *DevstackTimeoutSuite) TestRunningTimeout() {
	type TestCase struct {
		name           string
		nodeCount      int
		concurrency    int
		defaultTimeout time.Duration
		jobTimeout     time.Duration
		sleepTime      time.Duration
		completedCount int
		errorCount     int // when execution takes too long
	}

	runTest := func(testCase TestCase) {
		// required by job_timeout_greater_than_max_but_on_allowed_list
		namespace := ""
		testScenario := scenario.Scenario{
			Stack: &scenario.StackConfig{
				DevStackOptions: []devstack.ConfigOption{
					devstack.WithNumberOfHybridNodes(testCase.nodeCount),
					devstack.WithBacalhauConfigOverride(types.Bacalhau{
						Orchestrator: types.Orchestrator{
							Scheduler: types.Scheduler{
								HousekeepingInterval: types.Duration(100 * time.Millisecond),
								// we want compute nodes to fail first instead or requesters to cancel
								HousekeepingTimeout: types.Duration(2 * time.Second),
							},
						},
						JobDefaults: types.JobDefaults{
							Batch: types.BatchJobDefaultsConfig{
								Task: types.BatchTaskDefaultConfig{
									Timeouts: types.TaskTimeoutConfig{
										TotalTimeout: types.Duration(testCase.defaultTimeout),
									},
								},
							},
						},
					}),
				},
				ExecutorConfig: noop.ExecutorConfig{
					ExternalHooks: noop.ExecutorConfigExternalHooks{
						JobHandler: func(ctx context.Context, execContext noop.ExecutionContext) (*models.RunCommandResult, error) {
							time.Sleep(testCase.sleepTime)
							return executor.WriteJobResults(execContext.ResultsDir, strings.NewReader(""), strings.NewReader(""), 0, nil, executor.OutputLimits{
								MaxStdoutFileLength:   system.MaxStdoutFileLength,
								MaxStdoutReturnLength: system.MaxStdoutReturnLength,
								MaxStderrFileLength:   system.MaxStderrFileLength,
								MaxStderrReturnLength: system.MaxStderrReturnLength,
							}), nil
						},
					},
				},
			},
			Job: &models.Job{
				Name:      suite.T().Name(),
				Namespace: namespace,
				Type:      models.JobTypeBatch,
				Count:     testCase.concurrency,
				Tasks: []*models.Task{
					{
						Name: suite.T().Name(),
						Engine: &models.SpecConfig{
							Type:   models.EngineNoop,
							Params: make(map[string]interface{}),
						},
						Timeouts: &models.TimeoutConfig{
							TotalTimeout: int64(testCase.jobTimeout.Seconds()),
						},
					},
				},
			},
			JobCheckers: []scenario.StateChecks{
				scenario.WaitForExecutionStates(map[models.ExecutionStateType]int{
					models.ExecutionStateCompleted: testCase.completedCount,
					models.ExecutionStateFailed:    testCase.errorCount,
				}),
			},
		}

		suite.RunScenario(testScenario)
	}

	for _, testCase := range []TestCase{
		{
			name:           "sleep_within_default_timeout",
			defaultTimeout: 10 * time.Second,
			nodeCount:      1,
			concurrency:    1,
			sleepTime:      100 * time.Millisecond,
			completedCount: 1,
		},
		{
			name:           "sleep_within_defined_timeout",
			defaultTimeout: 20 * time.Second,
			nodeCount:      1,
			concurrency:    1,
			jobTimeout:     10 * time.Second,
			sleepTime:      100 * time.Millisecond,
			completedCount: 1,
		},
		{
			name:           "sleep_within_timeout_buffer",
			defaultTimeout: 20 * time.Second,
			nodeCount:      1,
			concurrency:    1,
			jobTimeout:     1 * time.Millisecond,
			sleepTime:      100 * time.Millisecond, // less than 500ms buffer
			completedCount: 1,
		},
		{
			name:           "sleep_longer_than_default_running_timeout",
			defaultTimeout: 1 * time.Second,
			nodeCount:      1,
			concurrency:    1,
			sleepTime:      20 * time.Second,
			errorCount:     1,
		},
		{
			name:           "sleep_longer_than_defined_running_timeout",
			defaultTimeout: 40 * time.Second,
			nodeCount:      1,
			concurrency:    1,
			sleepTime:      20 * time.Second,
			jobTimeout:     1 * time.Second,
			errorCount:     1,
		},
	} {
		suite.Run(testCase.name, func() {
			runTest(testCase)
		})
	}
}
