package types

import (
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

type ComputeConfig struct {
	Capacity           CapacityConfig           `yaml:"Capacity"`
	ExecutionStore     JobStoreConfig           `yaml:"ExecutionStore"`
	JobTimeouts        JobTimeoutConfig         `yaml:"JobTimeouts"`
	Queue              QueueConfig              `yaml:"Queue"`
	LoggingSensor      LoggingSensorConfig      `yaml:"Logging"`
	ManifestCache      DockerCacheConfig        `yaml:"ManifestCache"`
	LogStreamConfig    LogStreamConfig          `yaml:"LogStream"`
	LocalPublisher     LocalPublisherConfig     `yaml:"LocalPublisher"`
	Labels             LabelsConfig             `yaml:"Labels"`
	Executor           ExecutorConfig           `yaml:"Executor"`
	BufferedExecutor   BufferedExecutorConfig   `yaml:"BufferedExecutor"`
	JobSelection       JobSelectionPolicyConfig `yaml:"JobSelection"`
	StorageProviders   StorageProvidersConfig   `yaml:"StorageProviders"`
	ExecutorProviders  ExecutorProvidersConfig  `yaml:"ExecutorProviders"`
	PublisherProviders PublisherProvidersConfig `yaml:"PublisherProviders"`
}

type LabelsConfig struct {
	Labels map[string]string `yaml:"Labels"`
}

type ExecutorConfig struct {
	StorageDirectory string `yaml:"StorageDirectory"`
	ResultsPath      string `yaml:"ResultsPath"`
}

type BufferedExecutorConfig struct {
	DefaultJobExecutionTimeout Duration `yaml:"DefaultJobExecutionTimeout"`
}

type JobSelectionPolicyConfig struct {
	Policy model.JobSelectionPolicy `yaml:"Policy"`
}

type CapacityConfig struct {
	IgnorePhysicalResourceLimits bool `yaml:"IgnorePhysicalResourceLimits"`
	// Total amount of resource the system can be using at one time in aggregate for all jobs.
	TotalResourceLimits models.ResourcesConfig `yaml:"TotalResourceLimits"`
	// Per job amount of resource the system can be using at one time.
	JobResourceLimits        models.ResourcesConfig `yaml:"JobResourceLimits"`
	DefaultJobResourceLimits models.ResourcesConfig `yaml:"DefaultJobResourceLimits"`
	QueueResourceLimits      models.ResourcesConfig `yaml:"QueueResourceLimits"`
}

type JobTimeoutConfig struct {
	// JobExecutionTimeoutClientIDBypassList is the list of clients that are allowed to bypass the job execution timeout
	// check.
	JobExecutionTimeoutClientIDBypassList []string `yaml:"JobExecutionTimeoutClientIDBypassList"`
	// JobNegotiationTimeout default timeout value to hold a bid for a job
	JobNegotiationTimeout Duration `yaml:"JobNegotiationTimeout"`
	// MinJobExecutionTimeout default value for the minimum execution timeout this compute node supports. Jobs with
	// lower timeout requirements will not be bid on.
	MinJobExecutionTimeout Duration `yaml:"MinJobExecutionTimeout"`
	// MaxJobExecutionTimeout default value for the maximum execution timeout this compute node supports. Jobs with
	// higher timeout requirements will not be bid on.
	MaxJobExecutionTimeout Duration `yaml:"MaxJobExecutionTimeout"`
	// DefaultJobExecutionTimeout default value for the execution timeout this compute node will assign to jobs with
	// no timeout requirement defined.
	DefaultJobExecutionTimeout Duration `yaml:"DefaultJobExecutionTimeout"`
}

type QueueConfig struct {
}

type LoggingSensorConfig struct {
	// logging running executions
	LogRunningExecutionsInterval Duration `yaml:"LogRunningExecutionsInterval"`
}

type LogStreamConfig struct {
	// How many messages to buffer in the log stream channel, per stream
	ChannelBufferSize int `yaml:"ChannelBufferSize"`
}

type StorageProvidersConfig struct {
	AllowListedLocalPaths []string `yaml:"AllowListedLocalPaths"`
	Disabled              []string `yaml:"Disabled"`
}

type ExecutorProvidersConfig struct {
	Disabled []string `yaml:"Disabled"`
}

type PublisherProvidersConfig struct {
	Local    LocalPublisherConfig `yaml:"Local"`
	Disabled []string             `yaml:"Disabled"`
}

type LocalPublisherConfig struct {
	Address   string `yaml:"Address"`
	Port      int    `yaml:"Port"`
	Directory string `yaml:"Directory"`
}
