package types

import (
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

type ComputeConfig struct {
	Capacity             CapacityConfig            `yaml:"Capacity"`
	ExecutionStore       JobStoreConfig            `yaml:"ExecutionStore"`
	JobTimeouts          JobTimeoutConfig          `yaml:"JobTimeouts"`
	JobSelection         JobSelectionPolicyConfig  `yaml:"JobSelection"`
	Queue                QueueConfig               `yaml:"Queue"`
	Logging              LoggingConfig             `yaml:"Logging"`
	ManifestCache        DockerCacheConfig         `yaml:"ManifestCache"`
	LogStreamConfig      LogStreamConfig           `yaml:"LogStream"`
	LocalPublisher       LocalPublisherConfig      `yaml:"LocalPublisher"`
	ControlPlaneSettings ComputeControlPlaneConfig `yaml:"ClusterTimeouts"`

	DockerCredentials  DockerCredentialsConfig  `yaml:"DockerCredentials"`
	Executor           ExecutorConfig           `yaml:"Executor"`
	BufferedExecutor   BufferedExecutorConfig   `yaml:"BufferedExecutor"`
	Labels             LabelsConfig             `yaml:"Labels"`
	StorageProviders   StorageProvidersConfig   `yaml:"StorageProviders"`
	ExecutorProviders  ExecutorProvidersConfig  `yaml:"ExecutorProviders"`
	PublisherProviders PublisherProvidersConfig `yaml:"PublisherProviders"`
}

type CapacityConfig struct {
	IgnorePhysicalResourceLimits bool `yaml:"IgnorePhysicalResourceLimits"`

	// TODO(forrest) [refactor]: the models.ResourceConfig type should be defined in this package, not the models package

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

type LoggingConfig struct {
	// logging running executions
	LogRunningExecutionsInterval Duration `yaml:"LogRunningExecutionsInterval"`
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
	InfoUpdateFrequency Duration `yaml:"InfoUpdateFrequency"`

	// How often the compute node will send current resource availability to the requester node.
	ResourceUpdateFrequency Duration `yaml:"ResourceUpdateFrequency"`

	// How often the compute node will send a heartbeat to the requester node to let it know
	// that the compute node is still alive. This should be less than the requester's configured
	// heartbeat timeout to avoid flapping.
	HeartbeatFrequency Duration `yaml:"HeartbeatFrequency"`

	// This is the pubsub topic that the compute node will use to send heartbeats to the requester node.
	HeartbeatTopic string `yaml:"HeartbeatTopic"`
}

type JobSelectionPolicyConfig struct {
	// TODO(forrest) [refactor]: this type should be defined in this package, not the model package
	Policy model.JobSelectionPolicy `yaml:"Policy"`
}

type BufferedExecutorConfig struct {
	DefaultJobExecutionTimeout Duration `yaml:"DefaultJobExecutionTimeout"`
}

type ExecutorConfig struct {
	StorageDirectory string `yaml:"StorageDirectory"`
	ResultsPath      string `yaml:"ResultsPath"`
}

type LabelsConfig struct {
	Labels map[string]string `yaml:"Labels"`
}

type StorageProvidersConfig struct {
	AllowListedLocalPaths     []string `yaml:"AllowListedLocalPaths"`
	Disabled                  []string `yaml:"Disabled"`
	VolumeSizeRequestTimeout  Duration `yaml:"VolumeSizeRequestTimeout"`
	DownloadURLRequestRetries int      `yaml:"DownloadURLRequestRetries"`
	DownloadURLRequestTimeout Duration `yaml:"DownloadURLRequestTimeout"`
}

type ExecutorProvidersConfig struct {
	Disabled []string `yaml:"Disabled"`
}

type PublisherProvidersConfig struct {
	Local    LocalPublisherConfig `yaml:"Local"`
	Disabled []string             `yaml:"Disabled"`
}

type DockerCredentialsConfig struct {
	Username string
	Password string
}

// TODO(forrest) [correctness]: not sure I want to enforce this interface on the config this way, although it seems
// convenient for now
func (d *DockerCredentialsConfig) IsValid() bool {
	return d.Username != "" && d.Password != ""
}
