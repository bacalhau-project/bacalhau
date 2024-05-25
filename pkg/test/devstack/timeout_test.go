//go:build integration || !unit

package devstack

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	legacy_job "github.com/bacalhau-project/bacalhau/pkg/legacyjob"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator/transformer"

	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/models"

	"github.com/bacalhau-project/bacalhau/pkg/devstack"
	"github.com/bacalhau-project/bacalhau/pkg/executor"
	"github.com/bacalhau-project/bacalhau/pkg/executor/noop"
	_ "github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/node"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator/retry"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/bacalhau-project/bacalhau/pkg/test/scenario"
	testutils "github.com/bacalhau-project/bacalhau/pkg/test/utils"
)

type DevstackTimeoutSuite struct {
	scenario.ScenarioRunner
}

func TestDevstackTimeoutSuite(t *testing.T) {
	suite.Run(t, new(DevstackTimeoutSuite))
}

func (suite *DevstackTimeoutSuite) TestRunningTimeout() {
	type TestCase struct {
		name                                string
		nodeCount                           int
		concurrency                         int
		computeJobNegotiationTimeout        time.Duration
		computeJobExecutionBypassList       []string
		computeMinJobExecutionTimeout       time.Duration
		computeMaxJobExecutionTimeout       time.Duration
		requesterDefaultJobExecutionTimeout time.Duration
		jobTimeout                          time.Duration
		sleepTime                           time.Duration
		completedCount                      int
		rejectedCount                       int // when no bids are received
		errorCount                          int // when execution takes too long
	}

	runTest := func(testCase TestCase) {
		computeConfig, err := node.NewComputeConfigWith(node.ComputeConfigParams{
			JobNegotiationTimeout:                 testCase.computeJobNegotiationTimeout,
			MinJobExecutionTimeout:                testCase.computeMinJobExecutionTimeout,
			MaxJobExecutionTimeout:                testCase.computeMaxJobExecutionTimeout,
			JobExecutionTimeoutClientIDBypassList: testCase.computeJobExecutionBypassList,
		})
		suite.Require().NoError(err)

		requesterConfig, err := node.NewRequesterConfigWith(node.RequesterConfigParams{
			JobDefaults: transformer.JobDefaults{
				ExecutionTimeout: testCase.requesterDefaultJobExecutionTimeout,
			},
			HousekeepingBackgroundTaskInterval: 10 * time.Millisecond,
			HousekeepingTimeoutBuffer:          500 * time.Millisecond,
			RetryStrategy:                      retry.NewFixedStrategy(retry.FixedStrategyParams{ShouldRetry: false}),
		})
		suite.Require().NoError(err)

		testScenario := scenario.Scenario{
			Stack: &scenario.StackConfig{
				DevStackOptions: &devstack.DevStackOptions{NumberOfHybridNodes: testCase.nodeCount},
				ComputeConfig:   computeConfig,
				RequesterConfig: requesterConfig,
				ExecutorConfig: noop.ExecutorConfig{
					ExternalHooks: noop.ExecutorConfigExternalHooks{
						JobHandler: func(ctx context.Context, _ string, resultsDir string) (*models.RunCommandResult, error) {
							time.Sleep(testCase.sleepTime)
							return executor.WriteJobResults(resultsDir, strings.NewReader(""), strings.NewReader(""), 0, nil, executor.OutputLimits{
								MaxStdoutFileLength:   system.MaxStdoutFileLength,
								MaxStdoutReturnLength: system.MaxStdoutReturnLength,
								MaxStderrFileLength:   system.MaxStderrFileLength,
								MaxStderrReturnLength: system.MaxStderrReturnLength,
							}), nil
						},
					},
				},
			},
			Spec: testutils.MakeSpecWithOpts(suite.T(),
				legacy_job.WithPublisher(model.PublisherSpec{Type: model.PublisherIpfs}),
				legacy_job.WithTimeout(int64(testCase.jobTimeout.Seconds())),
			),
			Deal: model.Deal{
				Concurrency: testCase.concurrency,
			},
			JobCheckers: []legacy_job.CheckStatesFunction{
				legacy_job.WaitForExecutionStates(map[model.ExecutionStateType]int{
					model.ExecutionStateCompleted:         testCase.completedCount,
					model.ExecutionStateCancelled:         testCase.errorCount,
					model.ExecutionStateAskForBidRejected: testCase.rejectedCount,
				}),
			},
		}

		suite.RunScenario(testScenario)
	}

	clientID, err := config.GetClientID()
	suite.Require().NoError(err)
	for _, testCase := range []TestCase{
		{
			name:                                "sleep_within_default_timeout",
			computeJobNegotiationTimeout:        10 * time.Second,
			computeMinJobExecutionTimeout:       0 * time.Nanosecond,
			computeMaxJobExecutionTimeout:       1 * time.Minute,
			requesterDefaultJobExecutionTimeout: 10 * time.Second,
			nodeCount:                           1,
			concurrency:                         1,
			sleepTime:                           100 * time.Millisecond,
			completedCount:                      1,
		},
		{
			name:                                "sleep_within_defined_timeout",
			computeJobNegotiationTimeout:        10 * time.Second,
			computeMinJobExecutionTimeout:       1 * time.Nanosecond,
			computeMaxJobExecutionTimeout:       1 * time.Minute,
			requesterDefaultJobExecutionTimeout: 20 * time.Second,
			nodeCount:                           1,
			concurrency:                         1,
			jobTimeout:                          10 * time.Second,
			sleepTime:                           100 * time.Millisecond,
			completedCount:                      1,
		},
		{
			name:                                "sleep_within_timeout_buffer",
			computeJobNegotiationTimeout:        10 * time.Second,
			computeMinJobExecutionTimeout:       1 * time.Nanosecond,
			computeMaxJobExecutionTimeout:       1 * time.Minute,
			requesterDefaultJobExecutionTimeout: 20 * time.Second,
			nodeCount:                           1,
			concurrency:                         1,
			jobTimeout:                          1 * time.Millisecond,
			sleepTime:                           100 * time.Millisecond, // less than 500ms buffer
			completedCount:                      1,
		},
		{
			name:                                "sleep_longer_than_default_running_timeout",
			computeJobNegotiationTimeout:        10 * time.Second,
			computeMinJobExecutionTimeout:       1 * time.Nanosecond,
			computeMaxJobExecutionTimeout:       1 * time.Minute,
			requesterDefaultJobExecutionTimeout: 1 * time.Millisecond,
			nodeCount:                           1,
			concurrency:                         1,
			sleepTime:                           20 * time.Second,
			errorCount:                          1,
		},
		{
			name:                                "sleep_longer_than_defined_running_timeout",
			computeJobNegotiationTimeout:        10 * time.Second,
			computeMinJobExecutionTimeout:       1 * time.Nanosecond,
			computeMaxJobExecutionTimeout:       1 * time.Minute,
			requesterDefaultJobExecutionTimeout: 40 * time.Second,
			nodeCount:                           1,
			concurrency:                         1,
			sleepTime:                           20 * time.Second,
			jobTimeout:                          1 * time.Second,
			errorCount:                          1,
		},
		{
			// no bid will be submitted, so the requester node should time out
			name:                                "job_timeout_longer_than_max_running_timeout",
			computeJobNegotiationTimeout:        10 * time.Second,
			computeMinJobExecutionTimeout:       1 * time.Nanosecond,
			computeMaxJobExecutionTimeout:       1 * time.Minute,
			requesterDefaultJobExecutionTimeout: 40 * time.Second,
			nodeCount:                           1,
			concurrency:                         1,
			sleepTime:                           20 * time.Second,
			jobTimeout:                          2 * time.Minute,
			rejectedCount:                       1,
		},
		{
			// no bid will be submitted, so the requester node should time out
			name:                                "job_timeout_less_than_min_running_timeout",
			computeJobNegotiationTimeout:        10 * time.Second,
			computeMinJobExecutionTimeout:       5 * time.Minute,
			computeMaxJobExecutionTimeout:       10 * time.Minute,
			requesterDefaultJobExecutionTimeout: 40 * time.Second,
			nodeCount:                           1,
			concurrency:                         1,
			sleepTime:                           20 * time.Second,
			jobTimeout:                          2 * time.Minute,
			rejectedCount:                       1,
		},
		{
			name:                                "job_timeout_greater_than_max",
			computeJobNegotiationTimeout:        10 * time.Second,
			computeMinJobExecutionTimeout:       1 * time.Nanosecond,
			computeMaxJobExecutionTimeout:       1 * time.Minute,
			requesterDefaultJobExecutionTimeout: 40 * time.Second,
			nodeCount:                           1,
			concurrency:                         1,
			sleepTime:                           1 * time.Second,
			jobTimeout:                          2 * time.Minute,
			rejectedCount:                       1,
		},
		{
			name:                                "job_timeout_greater_than_max_but_on_allowed_list",
			computeJobExecutionBypassList:       []string{clientID},
			computeJobNegotiationTimeout:        10 * time.Second,
			computeMinJobExecutionTimeout:       1 * time.Nanosecond,
			computeMaxJobExecutionTimeout:       1 * time.Minute,
			requesterDefaultJobExecutionTimeout: 40 * time.Second,
			nodeCount:                           1,
			concurrency:                         1,
			sleepTime:                           1 * time.Second,
			jobTimeout:                          2 * time.Minute,
			completedCount:                      1,
		},
	} {
		suite.Run(testCase.name, func() {
			runTest(testCase)
		})
	}
}
