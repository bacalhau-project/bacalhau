package types

import "github.com/bacalhau-project/bacalhau/pkg/model"

type ComputeConfig struct {
	Capacity       CapacityConfig
	ExecutionStore StorageConfig
	JobTimeouts    JobTimeoutConfig
	JobSelection   model.JobSelectionPolicy
	Queue          QueueConfig
	Logging        LoggingConfig
}

type CapacityConfig struct {
	IgnorePhysicalResourceLimits bool
	TotalResourceLimits          model.ResourceUsageConfig
	JobResourceLimits            model.ResourceUsageConfig
	DefaultJobResourceLimits     model.ResourceUsageConfig
	QueueResourceLimits          model.ResourceUsageConfig
}

type JobTimeoutConfig struct {
	// JobExecutionTimeoutClientIDBypassList is the list of clients that are allowed to bypass the job execution timeout
	// check.
	JobExecutionTimeoutClientIDBypassList []string
	// JobNegotiationTimeout default timeout value to hold a bid for a job
	JobNegotiationTimeout Duration
	// MinJobExecutionTimeout default value for the minimum execution timeout this compute node supports. Jobs with
	// lower timeout requirements will not be bid on.
	MinJobExecutionTimeout Duration
	// MaxJobExecutionTimeout default value for the maximum execution timeout this compute node supports. Jobs with
	// higher timeout requirements will not be bid on.
	MaxJobExecutionTimeout Duration
	// DefaultJobExecutionTimeout default value for the execution timeout this compute node will assign to jobs with
	// no timeout requirement defined.
	DefaultJobExecutionTimeout Duration
}

type QueueConfig struct {
	// How long the buffer would backoff before polling the queue again for new jobs
	ExecutorBufferBackoffDuration Duration
}

type LoggingConfig struct {
	// logging running executions
	LogRunningExecutionsInterval Duration
}
