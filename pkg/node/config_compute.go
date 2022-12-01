package node

import (
	"context"
	"fmt"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/compute/capacity"
	"github.com/filecoin-project/bacalhau/pkg/model"
)

type ComputeConfigParams struct {
	// Capacity config
	TotalResourceLimits          model.ResourceUsageData
	JobResourceLimits            model.ResourceUsageData
	DefaultJobResourceLimits     model.ResourceUsageData
	PhysicalResourcesProvider    capacity.Provider
	OverCommitResourcesFactor    float64
	IgnorePhysicalResourceLimits bool

	// Timeout config
	JobNegotiationTimeout      time.Duration
	MinJobExecutionTimeout     time.Duration
	MaxJobExecutionTimeout     time.Duration
	DefaultJobExecutionTimeout time.Duration

	// Bid strategies config
	JobSelectionPolicy model.JobSelectionPolicy

	// logging running executions
	LogRunningExecutionsInterval time.Duration
}

type ComputeConfig struct {
	// Capacity config
	TotalResourceLimits          model.ResourceUsageData
	JobResourceLimits            model.ResourceUsageData
	DefaultJobResourceLimits     model.ResourceUsageData
	OverCommitResourcesFactor    float64
	IgnorePhysicalResourceLimits bool

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

	// Bid strategies config
	JobSelectionPolicy model.JobSelectionPolicy

	// logging running executions
	LogRunningExecutionsInterval time.Duration
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

	// Get available physical resources in the host
	physicalResourcesProvider := params.PhysicalResourcesProvider
	if physicalResourcesProvider == nil {
		physicalResourcesProvider = DefaultComputeConfig.PhysicalResourcesProvider
	}
	physicalResources, err := physicalResourcesProvider.AvailableCapacity(context.Background())
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

	// populate default job resource limits with default values and job resource limits if not set
	defaultJobResourceLimits := params.DefaultJobResourceLimits.
		Intersect(DefaultComputeConfig.DefaultJobResourceLimits)

	if params.OverCommitResourcesFactor == 0 {
		params.OverCommitResourcesFactor = DefaultComputeConfig.OverCommitResourcesFactor
	}

	config = ComputeConfig{
		TotalResourceLimits:          totalResourceLimits,
		JobResourceLimits:            jobResourceLimits,
		DefaultJobResourceLimits:     defaultJobResourceLimits,
		OverCommitResourcesFactor:    params.OverCommitResourcesFactor,
		IgnorePhysicalResourceLimits: params.IgnorePhysicalResourceLimits,

		JobNegotiationTimeout:      params.JobNegotiationTimeout,
		MinJobExecutionTimeout:     params.MinJobExecutionTimeout,
		MaxJobExecutionTimeout:     params.MaxJobExecutionTimeout,
		DefaultJobExecutionTimeout: params.DefaultJobExecutionTimeout,

		JobSelectionPolicy: params.JobSelectionPolicy,

		LogRunningExecutionsInterval: params.LogRunningExecutionsInterval,
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

	if !config.DefaultJobResourceLimits.LessThanEq(config.JobResourceLimits) {
		err = fmt.Errorf("default job resource limits %+v exceed job resource limits %+v",
			config.DefaultJobResourceLimits, config.JobResourceLimits)
		return
	}
}
