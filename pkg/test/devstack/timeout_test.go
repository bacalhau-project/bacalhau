//go:build !(unit && (windows || darwin))

package devstack

import (
	"fmt"
	"testing"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/devstack"
	"github.com/filecoin-project/bacalhau/pkg/requesternode"

	"github.com/filecoin-project/bacalhau/pkg/computenode"
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
		name                   string
		nodeCount              int
		minBids                int
		concurrency            int
		computeTimeoutConfig   computenode.ComputeTimeoutConfig
		requesterTimeoutConfig requesternode.RequesterTimeoutConfig
		jobTimeout             float64
		sleepSeconds           float32
		completedCount         int
		errorCount             int
	}

	runTest := func(testCase TestCase) {
		testScenario := scenario.Scenario{
			Stack: &scenario.StackConfig{
				DevStackOptions: &devstack.DevStackOptions{NumberOfNodes: testCase.nodeCount},
				ComputeNodeConfig: &computenode.ComputeNodeConfig{
					TimeoutConfig:                      testCase.computeTimeoutConfig,
					StateManagerBackgroundTaskInterval: 1 * time.Second,
				},
				RequesterNodeConfig: &requesternode.RequesterNodeConfig{
					TimeoutConfig:                      testCase.requesterTimeoutConfig,
					StateManagerBackgroundTaskInterval: 1 * time.Second,
				},
			},
			Spec: model.Spec{
				Engine:    model.EngineDocker,
				Verifier:  model.VerifierNoop,
				Publisher: model.PublisherNoop,
				Timeout:   testCase.jobTimeout,
				Docker: model.JobSpecDocker{
					Image: "ubuntu:latest",
					Entrypoint: []string{
						"sleep",
						fmt.Sprintf("%.3f", testCase.sleepSeconds),
					},
				},
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
			name: "sleep_within_default_timeout",
			computeTimeoutConfig: computenode.ComputeTimeoutConfig{
				JobNegotiationTimeout:  10 * time.Second,
				MinJobExecutionTimeout: 0 * time.Nanosecond,
				MaxJobExecutionTimeout: 1 * time.Minute},
			requesterTimeoutConfig: requesternode.RequesterTimeoutConfig{
				JobNegotiationTimeout:      10 * time.Second,
				DefaultJobExecutionTimeout: 10 * time.Second,
				MinJobExecutionTimeout:     1 * time.Nanosecond},
			nodeCount:      1,
			minBids:        1,
			concurrency:    1,
			sleepSeconds:   0.1,
			completedCount: 1,
		},
		{
			name: "sleep_within_defined_timeout",
			computeTimeoutConfig: computenode.ComputeTimeoutConfig{
				JobNegotiationTimeout:  10 * time.Second,
				MinJobExecutionTimeout: 0 * time.Nanosecond,
				MaxJobExecutionTimeout: 1 * time.Minute},
			requesterTimeoutConfig: requesternode.RequesterTimeoutConfig{
				JobNegotiationTimeout:      10 * time.Second,
				DefaultJobExecutionTimeout: 20 * time.Second,
				MinJobExecutionTimeout:     1 * time.Nanosecond},
			nodeCount:      1,
			minBids:        1,
			concurrency:    1,
			jobTimeout:     10,
			sleepSeconds:   0.1,
			completedCount: 1,
		},
		{
			name: "sleep_longer_than_default_running_timeout",
			computeTimeoutConfig: computenode.ComputeTimeoutConfig{
				JobNegotiationTimeout:  10 * time.Second,
				MinJobExecutionTimeout: 0 * time.Nanosecond,
				MaxJobExecutionTimeout: 1 * time.Minute},
			requesterTimeoutConfig: requesternode.RequesterTimeoutConfig{
				JobNegotiationTimeout:      10 * time.Second,
				DefaultJobExecutionTimeout: 1 * time.Millisecond,
				MinJobExecutionTimeout:     1 * time.Nanosecond},
			nodeCount:    1,
			minBids:      1,
			concurrency:  1,
			sleepSeconds: 20,
			errorCount:   1,
		},
		{
			name: "sleep_longer_than_defined_running_timeout",
			computeTimeoutConfig: computenode.ComputeTimeoutConfig{
				JobNegotiationTimeout:  10 * time.Second,
				MinJobExecutionTimeout: 0 * time.Nanosecond,
				MaxJobExecutionTimeout: 1 * time.Minute},
			requesterTimeoutConfig: requesternode.RequesterTimeoutConfig{
				JobNegotiationTimeout:      10 * time.Second,
				DefaultJobExecutionTimeout: 40 * time.Second,
				MinJobExecutionTimeout:     1 * time.Nanosecond},
			nodeCount:    1,
			minBids:      1,
			concurrency:  1,
			sleepSeconds: 20,
			jobTimeout:   0.001, // 1 millisecond
			errorCount:   1,
		},
		{
			// no bid will be submitted, so the requester node should timeout
			name: "job_timeout_longer_than_max_running_timeout",
			computeTimeoutConfig: computenode.ComputeTimeoutConfig{
				JobNegotiationTimeout:  10 * time.Second,
				MinJobExecutionTimeout: 0 * time.Nanosecond,
				MaxJobExecutionTimeout: 1 * time.Minute},
			requesterTimeoutConfig: requesternode.RequesterTimeoutConfig{
				JobNegotiationTimeout:      500 * time.Millisecond,
				DefaultJobExecutionTimeout: 40 * time.Second,
				MinJobExecutionTimeout:     0 * time.Nanosecond},
			nodeCount:    1,
			minBids:      1,
			concurrency:  1,
			sleepSeconds: 20,
			jobTimeout:   120, // 2 minutes
			errorCount:   1,
		},
		{
			// no bid will be submitted, so the requester node should timeout
			name: "job_timeout_less_than_min_running_timeout",
			computeTimeoutConfig: computenode.ComputeTimeoutConfig{
				JobNegotiationTimeout:  10 * time.Second,
				MinJobExecutionTimeout: 5 * time.Minute,
				MaxJobExecutionTimeout: 10 * time.Minute},
			requesterTimeoutConfig: requesternode.RequesterTimeoutConfig{
				JobNegotiationTimeout:      500 * time.Millisecond,
				DefaultJobExecutionTimeout: 40 * time.Second,
				MinJobExecutionTimeout:     0 * time.Nanosecond},
			nodeCount:    1,
			minBids:      1,
			concurrency:  1,
			sleepSeconds: 20,
			jobTimeout:   120, // 2 minutes
			errorCount:   1,
		},
		{
			name: "bid_timeout",
			computeTimeoutConfig: computenode.ComputeTimeoutConfig{
				JobNegotiationTimeout:  200 * time.Millisecond,
				MinJobExecutionTimeout: 0 * time.Nanosecond,
				MaxJobExecutionTimeout: 1 * time.Minute},
			requesterTimeoutConfig: requesternode.RequesterTimeoutConfig{
				JobNegotiationTimeout:      10 * time.Second,
				DefaultJobExecutionTimeout: 40 * time.Second,
				MinJobExecutionTimeout:     0 * time.Nanosecond},
			nodeCount:    1, // only one node is available
			minBids:      2, // but we need two bids, so compute node should timeout while waiting for its bid to be accepted
			concurrency:  1,
			sleepSeconds: 0.1,
			errorCount:   1,
		},
		{
			name: "verification_timeout",
			computeTimeoutConfig: computenode.ComputeTimeoutConfig{
				JobNegotiationTimeout:  10 * time.Second,
				MinJobExecutionTimeout: 0 * time.Nanosecond,
				MaxJobExecutionTimeout: 1 * time.Minute},
			requesterTimeoutConfig: requesternode.RequesterTimeoutConfig{
				JobNegotiationTimeout:      200 * time.Millisecond,
				DefaultJobExecutionTimeout: 40 * time.Second,
				MinJobExecutionTimeout:     0 * time.Nanosecond},
			nodeCount:    1, // only one node available
			minBids:      1,
			concurrency:  2, // but concurrency is 2, so requester should timeout on verification
			sleepSeconds: 0.1,
			errorCount:   1,
		},
	} {
		suite.Run(testCase.name, func() {
			runTest(testCase)
		})
	}
}
