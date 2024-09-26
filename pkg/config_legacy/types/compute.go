package types

import (
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

type ComputeConfig struct {
	Capacity             CapacityConfig            `yaml:"Capacity"`
	ExecutionStore       JobStoreConfig            `yaml:"ExecutionStore"`
	JobTimeouts          JobTimeoutConfig          `yaml:"JobTimeouts"`
	JobSelection         models.JobSelectionPolicy `yaml:"JobSelection"`
	Logging              LoggingConfig             `yaml:"Logging"`
	ManifestCache        DockerCacheConfig         `yaml:"ManifestCache"`
	LogStreamConfig      LogStreamConfig           `yaml:"LogStream"`
	LocalPublisher       LocalPublisherConfig      `yaml:"LocalPublisher"`
	ControlPlaneSettings ComputeControlPlaneConfig `yaml:"ClusterTimeouts"`
}

type CapacityConfig struct {
	IgnorePhysicalResourceLimits bool `yaml:"IgnorePhysicalResourceLimits"`
	// Total amount of resource the system can be using at one time in aggregate for all jobs.
	TotalResourceLimits models.ResourcesConfig `yaml:"TotalResourceLimits"`
	// Per job amount of resource the system can be using at one time.
	JobResourceLimits        models.ResourcesConfig `yaml:"JobResourceLimits"`
	DefaultJobResourceLimits models.ResourcesConfig `yaml:"DefaultJobResourceLimits"`
}

type JobTimeoutConfig struct {
	// JobExecutionTimeoutClientIDBypassList is the list of clients that are allowed to bypass the job execution timeout
	// check.
	JobExecutionTimeoutClientIDBypassList []string `yaml:"JobExecutionTimeoutClientIDBypassList"`
	// JobNegotiationTimeout default timeout value to hold a bid for a job
	JobNegotiationTimeout Duration `yaml:"JobNegotiationTimeout" swaggertype:"primitive,integer"`
	// MinJobExecutionTimeout default value for the minimum execution timeout this compute node supports. Jobs with
	// lower timeout requirements will not be bid on.
	MinJobExecutionTimeout Duration `yaml:"MinJobExecutionTimeout" swaggertype:"primitive,integer"`
	// MaxJobExecutionTimeout default value for the maximum execution timeout this compute node supports. Jobs with
	// higher timeout requirements will not be bid on.
	MaxJobExecutionTimeout Duration `yaml:"MaxJobExecutionTimeout" swaggertype:"primitive,integer"`
	// DefaultJobExecutionTimeout default value for the execution timeout this compute node will assign to jobs with
	// no timeout requirement defined.
	DefaultJobExecutionTimeout Duration `yaml:"DefaultJobExecutionTimeout" swaggertype:"primitive,integer"`
}

type LoggingConfig struct {
	// logging running executions
	LogRunningExecutionsInterval Duration `yaml:"LogRunningExecutionsInterval" swaggertype:"primitive,integer"`
}

type LogStreamConfig struct {
	// How many messages to buffer in the log stream channel, per stream
	ChannelBufferSize int `yaml:"ChannelBufferSize"`
}

type LocalPublisherConfig struct {
	Address   string `yaml:"Address"`
	Port      int    `yaml:"Port"`
	Directory string `yaml:"Directory"`
}

type ComputeControlPlaneConfig struct {
	// The frequency with which the compute node will send node info (inc current labels)
	// to the controlling requester node.
	InfoUpdateFrequency Duration `yaml:"InfoUpdateFrequency" swaggertype:"primitive,integer"`

	// How often the compute node will send current resource availability to the requester node.
	ResourceUpdateFrequency Duration `yaml:"ResourceUpdateFrequency" swaggertype:"primitive,integer"`

	// How often the compute node will send a heartbeat to the requester node to let it know
	// that the compute node is still alive. This should be less than the requester's configured
	// heartbeat timeout to avoid flapping.
	HeartbeatFrequency Duration `yaml:"HeartbeatFrequency" swaggertype:"primitive,integer"`

	// This is the pubsub topic that the compute node will use to send heartbeats to the requester node.
	HeartbeatTopic string `yaml:"HeartbeatTopic"`
}
