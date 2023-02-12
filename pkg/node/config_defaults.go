package node

import (
	"time"

	"github.com/filecoin-project/bacalhau/pkg/compute/capacity/system"
	"github.com/filecoin-project/bacalhau/pkg/model"
)

var DefaultComputeConfig = ComputeConfigParams{
	PhysicalResourcesProvider: system.NewPhysicalCapacityProvider(),
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
	NodeRankRandomnessRange:            10,
}
