package node

import (
	"runtime"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy/semantic"
	compute_system "github.com/bacalhau-project/bacalhau/pkg/compute/capacity/system"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator/transformer"
	"github.com/bacalhau-project/bacalhau/pkg/routing"
	"github.com/bacalhau-project/bacalhau/pkg/system"
)

var DefaultComputeConfig = ComputeConfigParams{
	PhysicalResourcesProvider: compute_system.NewPhysicalCapacityProvider(),
	DefaultJobResourceLimits: models.Resources{
		CPU:    0.1,               // 100m
		Memory: 100 * 1024 * 1024, // 100Mi
	},
	ExecutorBufferBackoffDuration: 50 * time.Millisecond,

	JobNegotiationTimeout:      3 * time.Minute,
	MinJobExecutionTimeout:     500 * time.Millisecond,
	MaxJobExecutionTimeout:     model.NoJobTimeout,
	DefaultJobExecutionTimeout: model.NoJobTimeout,

	LogRunningExecutionsInterval: 10 * time.Second,
	JobSelectionPolicy:           NewDefaultJobSelectionPolicy(),
}

var DefaultRequesterConfig = RequesterConfigParams{
	JobDefaults: transformer.JobDefaults{
		ExecutionTimeout: model.NoJobTimeout,
	},

	HousekeepingBackgroundTaskInterval: 30 * time.Second,
	NodeRankRandomnessRange:            5,
	OverAskForBidsFactor:               3,

	MinBacalhauVersion: models.BuildVersionInfo{
		Major: "1", Minor: "0", GitVersion: "v1.0.4",
	},

	EvalBrokerVisibilityTimeout:    60 * time.Second,
	EvalBrokerInitialRetryDelay:    1 * time.Second,
	EvalBrokerSubsequentRetryDelay: 30 * time.Second,
	EvalBrokerMaxRetryCount:        10,

	WorkerCount:                  runtime.NumCPU(),
	WorkerEvalDequeueTimeout:     5 * time.Second,
	WorkerEvalDequeueBaseBackoff: 1 * time.Second,
	WorkerEvalDequeueMaxBackoff:  30 * time.Second,
}

var TestRequesterConfig = RequesterConfigParams{
	JobDefaults: transformer.JobDefaults{
		ExecutionTimeout: 30 * time.Second,
	},
	HousekeepingBackgroundTaskInterval: 30 * time.Second,
	NodeRankRandomnessRange:            5,
	OverAskForBidsFactor:               3,

	MinBacalhauVersion: models.BuildVersionInfo{
		Major: "1", Minor: "0", GitVersion: "v1.0.4",
	},

	EvalBrokerVisibilityTimeout:    5 * time.Second,
	EvalBrokerInitialRetryDelay:    100 * time.Millisecond,
	EvalBrokerSubsequentRetryDelay: 100 * time.Millisecond,
	EvalBrokerMaxRetryCount:        3,

	WorkerCount:                  3,
	WorkerEvalDequeueTimeout:     200 * time.Millisecond,
	WorkerEvalDequeueBaseBackoff: 20 * time.Millisecond,
	WorkerEvalDequeueMaxBackoff:  200 * time.Millisecond,
}

func getRequesterConfigParams() RequesterConfigParams {
	if system.GetEnvironment() == system.EnvironmentTest {
		return TestRequesterConfig
	}
	return DefaultRequesterConfig
}

var DefaultNodeInfoPublishConfig = routing.NodeInfoPublisherIntervalConfig{
	Interval:             30 * time.Second,
	EagerPublishInterval: 5 * time.Second,
	EagerPublishDuration: 30 * time.Second,
}

// TestNodeInfoPublishConfig speeds up node announcements for tests
var TestNodeInfoPublishConfig = routing.NodeInfoPublisherIntervalConfig{
	Interval:             30 * time.Second,
	EagerPublishInterval: 10 * time.Millisecond,
	EagerPublishDuration: 5 * time.Second,
}

func GetNodeInfoPublishConfig() routing.NodeInfoPublisherIntervalConfig {
	if system.GetEnvironment() == system.EnvironmentTest {
		return TestNodeInfoPublishConfig
	}
	return DefaultNodeInfoPublishConfig
}

func NewDefaultJobSelectionPolicy() JobSelectionPolicy {
	return JobSelectionPolicy{
		Locality: semantic.Anywhere,
	}
}
