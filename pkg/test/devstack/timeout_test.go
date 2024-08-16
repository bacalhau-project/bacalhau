//go:build integration || !unit

package devstack

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	baccrypto "github.com/bacalhau-project/bacalhau/pkg/lib/crypto"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator/transformer"

	"github.com/bacalhau-project/bacalhau/pkg/devstack"
	"github.com/bacalhau-project/bacalhau/pkg/executor"
	"github.com/bacalhau-project/bacalhau/pkg/executor/noop"
	_ "github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/node"
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
		computeConfig, err := node.NewComputeConfigWith(suite.T().TempDir(), node.ComputeConfigParams{
			JobNegotiationTimeout:                 testCase.computeJobNegotiationTimeout,
			MinJobExecutionTimeout:                testCase.computeMinJobExecutionTimeout,
			MaxJobExecutionTimeout:                testCase.computeMaxJobExecutionTimeout,
			JobExecutionTimeoutClientIDBypassList: testCase.computeJobExecutionBypassList,
		})
		suite.Require().NoError(err)

		requesterConfig, err := node.NewRequesterConfigWith(node.RequesterConfigParams{
			JobDefaults: transformer.JobDefaults{
				TotalTimeout: testCase.requesterDefaultJobExecutionTimeout,
			},
			HousekeepingBackgroundTaskInterval: 100 * time.Millisecond,
			// we want compute nodes to fail first instead or requesters to cancel
			HousekeepingTimeoutBuffer: 2 * time.Second,
		})
		suite.Require().NoError(err)

		// required by job_timeout_greater_than_max_but_on_allowed_list
		namespace := ""
		if len(testCase.computeJobExecutionBypassList) > 0 {
			namespace = testCase.computeJobExecutionBypassList[0]
		}

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
					models.ExecutionStateCompleted:         testCase.completedCount,
					models.ExecutionStateFailed:            testCase.errorCount,
					models.ExecutionStateAskForBidRejected: testCase.rejectedCount,
				}),
			},
		}

		suite.RunScenario(testScenario)
	}

	userKey, err := baccrypto.LoadUserKey(suite.Config.UserKeyPath())
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
			requesterDefaultJobExecutionTimeout: 1 * time.Second,
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
			computeJobExecutionBypassList:       []string{userKey.ClientID()},
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
