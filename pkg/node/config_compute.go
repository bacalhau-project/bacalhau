package node

import (
	"context"
	"fmt"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy/semantic"
	"github.com/bacalhau-project/bacalhau/pkg/compute/capacity"
	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

// JobSelectionPolicy describe the rules for how a compute node selects an incoming job
type JobSelectionPolicy struct {
	// this describes if we should run a job based on
	// where the data is located - i.e. if the data is "local"
	// or if the data is "anywhere"
	Locality semantic.JobSelectionDataLocality `json:"locality"`
	// should we reject jobs that don't specify any data
	// the default is "accept"
	RejectStatelessJobs bool `json:"reject_stateless_jobs"`
	// should we accept jobs that specify networking
	// the default is "reject"
	AcceptNetworkedJobs bool `json:"accept_networked_jobs"`
	// external hooks that decide if we should take on the job or not
	// if either of these are given they will override the data locality settings
	ProbeHTTP string `json:"probe_http,omitempty"`
	ProbeExec string `json:"probe_exec,omitempty"`
}

type ComputeConfigParams struct {
	// Capacity config
	TotalResourceLimits          models.Resources
	QueueResourceLimits          models.Resources
	JobResourceLimits            models.Resources
	DefaultJobResourceLimits     models.Resources
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
	JobSelectionPolicy JobSelectionPolicy

	// logging running executions
	LogRunningExecutionsInterval time.Duration

	FailureInjectionConfig model.FailureInjectionComputeConfig

	BidSemanticStrategy bidstrategy.SemanticBidStrategy

	BidResourceStrategy bidstrategy.ResourceBidStrategy
}

type ComputeConfig struct {
	// Capacity config
	TotalResourceLimits          models.Resources
	QueueResourceLimits          models.Resources
	JobResourceLimits            models.Resources
	DefaultJobResourceLimits     models.Resources
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
	JobSelectionPolicy JobSelectionPolicy

	// logging running executions
	LogRunningExecutionsInterval time.Duration

	FailureInjectionConfig model.FailureInjectionComputeConfig

	BidSemanticStrategy bidstrategy.SemanticBidStrategy

	BidResourceStrategy bidstrategy.ResourceBidStrategy

	ExecutionStore store.ExecutionStore
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
		Merge(DefaultComputeConfig.TotalResourceLimits).
		Merge(physicalResources)

	// populate job resource limits with default values and total resource limits if not set
	jobResourceLimits := params.JobResourceLimits.
		Merge(DefaultComputeConfig.JobResourceLimits).
		Merge(*totalResourceLimits)

	// by default set the queue size to the total resource limits, which allows the node overcommit to double of the total resource limits.
	// i.e. total resource limits can be busy in running state, and enqueue up to the total resource limits in the queue.
	if params.QueueResourceLimits.IsZero() {
		params.QueueResourceLimits = *totalResourceLimits
	}

	// populate default job resource limits with default values and job resource limits if not set
	defaultJobResourceLimits := params.DefaultJobResourceLimits.
		Merge(DefaultComputeConfig.DefaultJobResourceLimits)

	config = ComputeConfig{
		TotalResourceLimits:           *totalResourceLimits,
		QueueResourceLimits:           params.QueueResourceLimits,
		JobResourceLimits:             *jobResourceLimits,
		DefaultJobResourceLimits:      *defaultJobResourceLimits,
		IgnorePhysicalResourceLimits:  params.IgnorePhysicalResourceLimits,
		ExecutorBufferBackoffDuration: params.ExecutorBufferBackoffDuration,

		JobNegotiationTimeout:      params.JobNegotiationTimeout,
		MinJobExecutionTimeout:     params.MinJobExecutionTimeout,
		MaxJobExecutionTimeout:     params.MaxJobExecutionTimeout,
		DefaultJobExecutionTimeout: params.DefaultJobExecutionTimeout,

		JobExecutionTimeoutClientIDBypassList: params.JobExecutionTimeoutClientIDBypassList,

		JobSelectionPolicy: params.JobSelectionPolicy,

		LogRunningExecutionsInterval: params.LogRunningExecutionsInterval,
		FailureInjectionConfig:       params.FailureInjectionConfig,
		BidSemanticStrategy:          params.BidSemanticStrategy,
		BidResourceStrategy:          params.BidResourceStrategy,
	}

	validateConfig(config, physicalResources)
	return config
}

func validateConfig(config ComputeConfig, physicalResources models.Resources) {
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
