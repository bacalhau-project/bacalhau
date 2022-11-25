//go:build integration || !unit

package devstack

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/devstack"
	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/executor/noop"
	"github.com/filecoin-project/bacalhau/pkg/node"
	"github.com/filecoin-project/bacalhau/pkg/requesternode"

	"github.com/filecoin-project/bacalhau/pkg/job"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/model"
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
		name                          string
		nodeCount                     int
		minBids                       int
		concurrency                   int
		computeJobNegotiationTimeout  time.Duration
		computeMinJobExecutionTimeout time.Duration
		computeMaxJobExecutionTimeout time.Duration
		requesterTimeoutConfig        requesternode.RequesterTimeoutConfig
		jobTimeout                    time.Duration
		sleepTime                     time.Duration
		completedCount                int
		errorCount                    int
	}

	runTest := func(testCase TestCase) {
		testScenario := scenario.Scenario{
			Stack: &scenario.StackConfig{
				DevStackOptions: &devstack.DevStackOptions{NumberOfNodes: testCase.nodeCount},
				ComputeConfig: node.NewComputeConfigWith(node.ComputeConfigParams{
					JobNegotiationTimeout:  testCase.computeJobNegotiationTimeout,
					MinJobExecutionTimeout: testCase.computeMinJobExecutionTimeout,
					MaxJobExecutionTimeout: testCase.computeMaxJobExecutionTimeout,
				}),
				RequesterNodeConfig: &requesternode.RequesterNodeConfig{
					TimeoutConfig:                      testCase.requesterTimeoutConfig,
					StateManagerBackgroundTaskInterval: 1 * time.Second,
				},
				ExecutorConfig: &noop.ExecutorConfig{
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
				job.WaitForJobStates(map[model.JobStateType]int{
					model.JobStateCompleted: testCase.completedCount,
					model.JobStateError:     testCase.errorCount,
				}),
			},
		}

		suite.RunScenario(testScenario)
	}

	for _, testCase := range []TestCase{
		{
			name:                          "sleep_within_default_timeout",
			computeJobNegotiationTimeout:  10 * time.Second,
			computeMinJobExecutionTimeout: 0 * time.Nanosecond,
			computeMaxJobExecutionTimeout: 1 * time.Minute,
			requesterTimeoutConfig: requesternode.RequesterTimeoutConfig{
				JobNegotiationTimeout:      10 * time.Second,
				DefaultJobExecutionTimeout: 10 * time.Second,
				MinJobExecutionTimeout:     1 * time.Nanosecond},
			nodeCount:      1,
			minBids:        1,
			concurrency:    1,
			sleepTime:      100 * time.Millisecond,
			completedCount: 1,
		},
		{
			name:                          "sleep_within_defined_timeout",
			computeJobNegotiationTimeout:  10 * time.Second,
			computeMinJobExecutionTimeout: 0 * time.Nanosecond,
			computeMaxJobExecutionTimeout: 1 * time.Minute,
			requesterTimeoutConfig: requesternode.RequesterTimeoutConfig{
				JobNegotiationTimeout:      10 * time.Second,
				DefaultJobExecutionTimeout: 20 * time.Second,
				MinJobExecutionTimeout:     1 * time.Nanosecond},
			nodeCount:      1,
			minBids:        1,
			concurrency:    1,
			jobTimeout:     10 * time.Second,
			sleepTime:      100 * time.Millisecond,
			completedCount: 1,
		},
		{
			name:                          "sleep_longer_than_default_running_timeout",
			computeJobNegotiationTimeout:  10 * time.Second,
			computeMinJobExecutionTimeout: 0 * time.Nanosecond,
			computeMaxJobExecutionTimeout: 1 * time.Minute,
			requesterTimeoutConfig: requesternode.RequesterTimeoutConfig{
				JobNegotiationTimeout:      10 * time.Second,
				DefaultJobExecutionTimeout: 1 * time.Millisecond,
				MinJobExecutionTimeout:     1 * time.Nanosecond},
			nodeCount:   1,
			minBids:     1,
			concurrency: 1,
			sleepTime:   20 * time.Second,
			errorCount:  1,
		},
		{
			name:                          "sleep_longer_than_defined_running_timeout",
			computeJobNegotiationTimeout:  10 * time.Second,
			computeMinJobExecutionTimeout: 0 * time.Nanosecond,
			computeMaxJobExecutionTimeout: 1 * time.Minute,
			requesterTimeoutConfig: requesternode.RequesterTimeoutConfig{
				JobNegotiationTimeout:      10 * time.Second,
				DefaultJobExecutionTimeout: 40 * time.Second,
				MinJobExecutionTimeout:     1 * time.Nanosecond},
			nodeCount:   1,
			minBids:     1,
			concurrency: 1,
			sleepTime:   20 * time.Second,
			jobTimeout:  1 * time.Millisecond,
			errorCount:  1,
		},
		{
			// no bid will be submitted, so the requester node should timeout
			name:                          "job_timeout_longer_than_max_running_timeout",
			computeJobNegotiationTimeout:  10 * time.Second,
			computeMinJobExecutionTimeout: 0 * time.Nanosecond,
			computeMaxJobExecutionTimeout: 1 * time.Minute,
			requesterTimeoutConfig: requesternode.RequesterTimeoutConfig{
				JobNegotiationTimeout:      500 * time.Millisecond,
				DefaultJobExecutionTimeout: 40 * time.Second,
				MinJobExecutionTimeout:     0 * time.Nanosecond},
			nodeCount:   1,
			minBids:     1,
			concurrency: 1,
			sleepTime:   20 * time.Second,
			jobTimeout:  2 * time.Minute,
			errorCount:  1,
		},
		{
			// no bid will be submitted, so the requester node should timeout
			name:                          "job_timeout_less_than_min_running_timeout",
			computeJobNegotiationTimeout:  10 * time.Second,
			computeMinJobExecutionTimeout: 5 * time.Minute,
			computeMaxJobExecutionTimeout: 10 * time.Minute,
			requesterTimeoutConfig: requesternode.RequesterTimeoutConfig{
				JobNegotiationTimeout:      500 * time.Millisecond,
				DefaultJobExecutionTimeout: 40 * time.Second,
				MinJobExecutionTimeout:     0 * time.Nanosecond},
			nodeCount:   1,
			minBids:     1,
			concurrency: 1,
			sleepTime:   20 * time.Second,
			jobTimeout:  2 * time.Minute,
			errorCount:  1,
		},
		{
			name:                          "bid_timeout",
			computeJobNegotiationTimeout:  200 * time.Millisecond,
			computeMinJobExecutionTimeout: 0 * time.Nanosecond,
			computeMaxJobExecutionTimeout: 1 * time.Minute,
			requesterTimeoutConfig: requesternode.RequesterTimeoutConfig{
				JobNegotiationTimeout:      10 * time.Second,
				DefaultJobExecutionTimeout: 40 * time.Second,
				MinJobExecutionTimeout:     0 * time.Nanosecond},
			nodeCount:   1, // only one node is available
			minBids:     2, // but we need two bids, so compute node should timeout while waiting for its bid to be accepted
			concurrency: 1,
			sleepTime:   100 * time.Millisecond,
			errorCount:  1,
		},
		{
			name:                          "verification_timeout",
			computeJobNegotiationTimeout:  10 * time.Second,
			computeMinJobExecutionTimeout: 0 * time.Nanosecond,
			computeMaxJobExecutionTimeout: 1 * time.Minute,
			requesterTimeoutConfig: requesternode.RequesterTimeoutConfig{
				JobNegotiationTimeout:      200 * time.Millisecond,
				DefaultJobExecutionTimeout: 40 * time.Second,
				MinJobExecutionTimeout:     0 * time.Nanosecond},
			nodeCount:   1, // only one node available
			minBids:     1,
			concurrency: 2, // but concurrency is 2, so requester should timeout on verification
			sleepTime:   100 * time.Millisecond,
			errorCount:  1,
		},
	} {
		suite.Run(testCase.name, func() {
			runTest(testCase)
		})
	}
}
