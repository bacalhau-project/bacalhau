//go:build integration

package devstack

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/devstack"
	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/executor/noop"
	"github.com/filecoin-project/bacalhau/pkg/job"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/node"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/test/scenario"
	"github.com/stretchr/testify/suite"
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
		minBids                             int
		concurrency                         int
		computeJobNegotiationTimeout        time.Duration
		computeJobExecutionBypassList       []string
		computeMinJobExecutionTimeout       time.Duration
		computeMaxJobExecutionTimeout       time.Duration
		requesterMinJobExecutionTimeout     time.Duration
		requesterDefaultJobExecutionTimeout time.Duration
		jobTimeout                          time.Duration
		sleepTime                           time.Duration
		completedCount                      int
		rejectedCount                       int // when no bids are received
		errorCount                          int // when execution takes too long
	}

	runTest := func(testCase TestCase) {
		testScenario := scenario.Scenario{
			Stack: &scenario.StackConfig{
				DevStackOptions: &devstack.DevStackOptions{NumberOfHybridNodes: testCase.nodeCount},
				ComputeConfig: node.NewComputeConfigWith(node.ComputeConfigParams{
					JobNegotiationTimeout:                 testCase.computeJobNegotiationTimeout,
					MinJobExecutionTimeout:                testCase.computeMinJobExecutionTimeout,
					MaxJobExecutionTimeout:                testCase.computeMaxJobExecutionTimeout,
					JobExecutionTimeoutClientIDBypassList: testCase.computeJobExecutionBypassList,
				}),
				RequesterConfig: node.NewRequesterConfigWith(node.RequesterConfigParams{
					MinJobExecutionTimeout:             testCase.requesterMinJobExecutionTimeout,
					DefaultJobExecutionTimeout:         testCase.requesterDefaultJobExecutionTimeout,
					HousekeepingBackgroundTaskInterval: 1 * time.Second,
				}),
				ExecutorConfig: noop.ExecutorConfig{
					ExternalHooks: noop.ExecutorConfigExternalHooks{
						JobHandler: func(ctx context.Context, shard model.JobShard, resultsDir string) (*model.RunCommandResult, error) {
							time.Sleep(testCase.sleepTime)
							return executor.WriteJobResults(resultsDir, strings.NewReader(""), strings.NewReader(""), 0, nil)
						},
					},
				},
			},
			Spec: model.Spec{
				Engine:    model.EngineNoop,
				Verifier:  model.VerifierNoop,
				Publisher: model.PublisherNoop,
				Timeout:   testCase.jobTimeout.Seconds(),
			},
			Deal: model.Deal{
				Concurrency: testCase.concurrency,
				MinBids:     testCase.minBids,
			},
			JobCheckers: []job.CheckStatesFunction{
				job.WaitForExecutionStates(map[model.ExecutionStateType]int{
					model.ExecutionStateCompleted:         testCase.completedCount,
					model.ExecutionStateFailed:            testCase.errorCount,
					model.ExecutionStateAskForBidRejected: testCase.rejectedCount,
				}),
			},
		}

		suite.RunScenario(testScenario)
	}

	for _, testCase := range []TestCase{
		{
			name:                                "sleep_within_default_timeout",
			computeJobNegotiationTimeout:        10 * time.Second,
			computeMinJobExecutionTimeout:       0 * time.Nanosecond,
			computeMaxJobExecutionTimeout:       1 * time.Minute,
			requesterDefaultJobExecutionTimeout: 10 * time.Second,
			requesterMinJobExecutionTimeout:     1 * time.Nanosecond,
			nodeCount:                           1,
			minBids:                             1,
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
			requesterMinJobExecutionTimeout:     1 * time.Nanosecond,
			nodeCount:                           1,
			minBids:                             1,
			concurrency:                         1,
			jobTimeout:                          10 * time.Second,
			sleepTime:                           100 * time.Millisecond,
			completedCount:                      1,
		},
		{
			name:                                "sleep_longer_than_default_running_timeout",
			computeJobNegotiationTimeout:        10 * time.Second,
			computeMinJobExecutionTimeout:       1 * time.Nanosecond,
			computeMaxJobExecutionTimeout:       1 * time.Minute,
			requesterDefaultJobExecutionTimeout: 1 * time.Millisecond,
			requesterMinJobExecutionTimeout:     1 * time.Nanosecond,
			nodeCount:                           1,
			minBids:                             1,
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
			requesterMinJobExecutionTimeout:     1 * time.Nanosecond,
			nodeCount:                           1,
			minBids:                             1,
			concurrency:                         1,
			sleepTime:                           20 * time.Second,
			jobTimeout:                          1 * time.Millisecond,
			errorCount:                          1,
		},
		{
			// no bid will be submitted, so the requester node should time out
			name:                                "job_timeout_longer_than_max_running_timeout",
			computeJobNegotiationTimeout:        10 * time.Second,
			computeMinJobExecutionTimeout:       1 * time.Nanosecond,
			computeMaxJobExecutionTimeout:       1 * time.Minute,
			requesterDefaultJobExecutionTimeout: 40 * time.Second,
			requesterMinJobExecutionTimeout:     1 * time.Nanosecond,
			nodeCount:                           1,
			minBids:                             1,
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
			requesterMinJobExecutionTimeout:     1 * time.Nanosecond,
			nodeCount:                           1,
			minBids:                             1,
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
			requesterMinJobExecutionTimeout:     1 * time.Nanosecond,
			nodeCount:                           1,
			minBids:                             1,
			concurrency:                         1,
			sleepTime:                           1 * time.Second,
			jobTimeout:                          2 * time.Minute,
			rejectedCount:                       1,
		},
		{
			name:                                "job_timeout_greater_than_max_but_on_allowed_list",
			computeJobExecutionBypassList:       []string{system.GetClientID()},
			computeJobNegotiationTimeout:        10 * time.Second,
			computeMinJobExecutionTimeout:       1 * time.Nanosecond,
			computeMaxJobExecutionTimeout:       1 * time.Minute,
			requesterDefaultJobExecutionTimeout: 40 * time.Second,
			requesterMinJobExecutionTimeout:     1 * time.Nanosecond,
			nodeCount:                           1,
			minBids:                             1,
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
