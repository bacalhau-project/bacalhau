package node

import (
	"context"
	"fmt"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/compute/capacity"
	"github.com/bacalhau-project/bacalhau/pkg/model"
)

type ComputeConfigParams struct {
	// Capacity config
	TotalResourceLimits          model.ResourceUsageData
	QueueResourceLimits          model.ResourceUsageData
	JobResourceLimits            model.ResourceUsageData
	DefaultJobResourceLimits     model.ResourceUsageData
	PhysicalResourcesProvider    capacity.Provider
	IgnorePhysicalResourceLimits bool

	ExecutorBufferBackoffDuration time.Duration

	// Timeout config
	JobNegotiationTimeout      time.Duration
	MinJobExecutionTimeout     time.Duration
	MaxJobExecutionTimeout     time.Duration
	DefaultJobExecutionTimeout time.Duration

	JobExecutionTimeoutClientIDBypassList []string

	// Bid strategies config
	JobSelectionPolicy model.JobSelectionPolicy

	// logging running executions
	LogRunningExecutionsInterval time.Duration

	SimulatorConfig model.SimulatorConfigCompute

	BidSemanticStrategy bidstrategy.SemanticBidStrategy

	BidResourceStrategy bidstrategy.ResourceBidStrategy
}

type ComputeConfig struct {
	// Capacity config
	TotalResourceLimits          model.ResourceUsageData
	QueueResourceLimits          model.ResourceUsageData
	JobResourceLimits            model.ResourceUsageData
	DefaultJobResourceLimits     model.ResourceUsageData
	IgnorePhysicalResourceLimits bool

	// How long the buffer would backoff before polling the queue again for new jobs
	ExecutorBufferBackoffDuration time.Duration

	// JobNegotiationTimeout default timeout value to hold a bid for a job
	JobNegotiationTimeout time.Duration
	// MinJobExecutionTimeout default value for the minimum execution timeout this compute node supports. Jobs with
	// lower timeout requirements will not be bid on.
	MinJobExecutionTimeout time.Duration
	// MaxJobExecutionTimeout default value for the maximum execution timeout this compute node supports. Jobs with
	// higher timeout requirements will not be bid on.
	MaxJobExecutionTimeout time.Duration
	// DefaultJobExecutionTimeout default value for the execution timeout this compute node will assign to jobs with
	// no timeout requirement defined.
	DefaultJobExecutionTimeout time.Duration

	// JobExecutionTimeoutClientIDBypassList is the list of clients that are allowed to bypass the job execution timeout
	// check.
	JobExecutionTimeoutClientIDBypassList []string

	// Bid strategies config
	JobSelectionPolicy model.JobSelectionPolicy

	// logging running executions
	LogRunningExecutionsInterval time.Duration

	SimulatorConfig model.SimulatorConfigCompute

	BidSemanticStrategy bidstrategy.SemanticBidStrategy

	BidResourceStrategy bidstrategy.ResourceBidStrategy
}

func NewComputeConfigWithDefaults() ComputeConfig {
	return NewComputeConfigWith(DefaultComputeConfig)
}

func NewComputeConfigWith(params ComputeConfigParams) (config ComputeConfig) {
	var err error

	defer func() {
		if err != nil {
			panic(fmt.Sprintf("Failed to initialize compute config %s", err.Error()))
		}
	}()
	if params.JobNegotiationTimeout == 0 {
		params.JobNegotiationTimeout = DefaultComputeConfig.JobNegotiationTimeout
	}
	if params.MinJobExecutionTimeout == 0 {
		params.MinJobExecutionTimeout = DefaultComputeConfig.MinJobExecutionTimeout
	}
	if params.MaxJobExecutionTimeout == 0 {
		params.MaxJobExecutionTimeout = DefaultComputeConfig.MaxJobExecutionTimeout
	}
	if params.DefaultJobExecutionTimeout == 0 {
		params.DefaultJobExecutionTimeout = DefaultComputeConfig.DefaultJobExecutionTimeout
	}
	if params.LogRunningExecutionsInterval == 0 {
		params.LogRunningExecutionsInterval = DefaultComputeConfig.LogRunningExecutionsInterval
	}
	if params.ExecutorBufferBackoffDuration == 0 {
		params.ExecutorBufferBackoffDuration = DefaultComputeConfig.ExecutorBufferBackoffDuration
	}

	// Get available physical resources in the host
	physicalResourcesProvider := params.PhysicalResourcesProvider
	if physicalResourcesProvider == nil {
		physicalResourcesProvider = DefaultComputeConfig.PhysicalResourcesProvider
	}
	physicalResources, err := physicalResourcesProvider.GetAvailableCapacity(context.Background())
	if err != nil {
		return
	}
	// populate total resource limits with default values and physical resources if not set
	totalResourceLimits := params.TotalResourceLimits.
		Intersect(DefaultComputeConfig.TotalResourceLimits).
		Intersect(physicalResources)

	// populate job resource limits with default values and total resource limits if not set
	jobResourceLimits := params.JobResourceLimits.
		Intersect(DefaultComputeConfig.JobResourceLimits).
		Intersect(totalResourceLimits)

	// by default set the queue size to the total resource limits, which allows the node overcommit to double of the total resource limits.
	// i.e. total resource limits can be busy in running state, and enqueue up to the total resource limits in the queue.
	if params.QueueResourceLimits.IsZero() {
		params.QueueResourceLimits = totalResourceLimits
	}

	// populate default job resource limits with default values and job resource limits if not set
	defaultJobResourceLimits := params.DefaultJobResourceLimits.
		Intersect(DefaultComputeConfig.DefaultJobResourceLimits)

	config = ComputeConfig{
		TotalResourceLimits:           totalResourceLimits,
		QueueResourceLimits:           params.QueueResourceLimits,
		JobResourceLimits:             jobResourceLimits,
		DefaultJobResourceLimits:      defaultJobResourceLimits,
		IgnorePhysicalResourceLimits:  params.IgnorePhysicalResourceLimits,
		ExecutorBufferBackoffDuration: params.ExecutorBufferBackoffDuration,

		JobNegotiationTimeout:      params.JobNegotiationTimeout,
		MinJobExecutionTimeout:     params.MinJobExecutionTimeout,
		MaxJobExecutionTimeout:     params.MaxJobExecutionTimeout,
		DefaultJobExecutionTimeout: params.DefaultJobExecutionTimeout,

		JobExecutionTimeoutClientIDBypassList: params.JobExecutionTimeoutClientIDBypassList,

		JobSelectionPolicy: params.JobSelectionPolicy,

		LogRunningExecutionsInterval: params.LogRunningExecutionsInterval,
		SimulatorConfig:              params.SimulatorConfig,
		BidSemanticStrategy:          params.BidSemanticStrategy,
		BidResourceStrategy:          params.BidResourceStrategy,
	}

	validateConfig(config, physicalResources)
	return config
}

func validateConfig(config ComputeConfig, physicalResources model.ResourceUsageData) {
	var err error
	defer func() {
		if err != nil {
			panic(fmt.Sprintf("Failed to validate compute config %s", err.Error()))
		}
	}()

	if !config.IgnorePhysicalResourceLimits && !config.TotalResourceLimits.LessThanEq(physicalResources) {
		err = fmt.Errorf("total resource limits %+v exceed physical resources %+v", config.TotalResourceLimits, physicalResources)
		return
	}

	if !config.JobResourceLimits.LessThanEq(config.TotalResourceLimits) {
		err = fmt.Errorf("job resource limits %+v exceed total resource limits %+v", config.JobResourceLimits, config.TotalResourceLimits)
		return
	}

	if !config.JobResourceLimits.LessThanEq(config.QueueResourceLimits) {
		err = fmt.Errorf("job resource limits %+v exceed queue size limits %+v, which will prevent processing the job",
			config.JobResourceLimits, config.QueueResourceLimits)
		return
	}

	if !config.DefaultJobResourceLimits.LessThanEq(config.JobResourceLimits) {
		err = fmt.Errorf("default job resource limits %+v exceed job resource limits %+v",
			config.DefaultJobResourceLimits, config.JobResourceLimits)
		return
	}
}
