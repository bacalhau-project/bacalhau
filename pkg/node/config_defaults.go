package node

import (
	"time"

	compute_system "github.com/bacalhau-project/bacalhau/pkg/compute/capacity/system"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/routing"
	"github.com/bacalhau-project/bacalhau/pkg/system"
)

var DefaultComputeConfig = ComputeConfigParams{
	PhysicalResourcesProvider: compute_system.NewPhysicalCapacityProvider(),
	DefaultJobResourceLimits: model.ResourceUsageData{
		CPU:    0.1,               // 100m
		Memory: 100 * 1024 * 1024, // 100Mi
	},
	ExecutorBufferBackoffDuration: 50 * time.Millisecond,

	JobNegotiationTimeout:      3 * time.Minute,
	MinJobExecutionTimeout:     500 * time.Millisecond,
	MaxJobExecutionTimeout:     60 * time.Minute,
	DefaultJobExecutionTimeout: 10 * time.Minute,

	LogRunningExecutionsInterval: 10 * time.Second,
}

var DefaultRequesterConfig = RequesterConfigParams{
	MinJobExecutionTimeout:     0 * time.Second,
	DefaultJobExecutionTimeout: 30 * time.Minute,

	HousekeepingBackgroundTaskInterval: 30 * time.Second,
	NodeRankRandomnessRange:            5,
	OverAskForBidsFactor:               3,

	MinBacalhauVersion: model.BuildVersionInfo{
		Major: "0", Minor: "3", GitVersion: "v0.3.26",
	},
}

var DefaultNodeInfoPublishConfig = routing.NodeInfoPublisherIntervalConfig{
	Interval:             30 * time.Second,
	EagerPublishInterval: 5 * time.Second,
	EagerPublishDuration: 30 * time.Second,
}

// speed up node announcements for tests
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
