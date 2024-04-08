package types

import (
	"github.com/bacalhau-project/bacalhau/pkg/model"
)

type RequesterConfig struct {
	JobDefaults JobDefaults `yaml:"JobDefaults"`
	// URL where to send external verification requests to.
	ExternalVerifierHook string `yaml:"ExternalVerifierHook"`
	// How the node decides what jobs to run.
	JobSelectionPolicy model.JobSelectionPolicy `yaml:"JobSelectionPolicy"`
	JobStore           JobStoreConfig           `yaml:"JobStore"`

	HousekeepingBackgroundTaskInterval Duration                              `yaml:"HousekeepingBackgroundTaskInterval"`
	NodeRankRandomnessRange            int                                   `yaml:"NodeRankRandomnessRange"`
	OverAskForBidsFactor               uint                                  `yaml:"OverAskForBidsFactor"`
	FailureInjectionConfig             model.FailureInjectionRequesterConfig `yaml:"FailureInjectionConfig"`

	TranslationEnabled bool `yaml:"TranslationEnabled"`

	EvaluationBroker EvaluationBrokerConfig `yaml:"EvaluationBroker"`
	Worker           WorkerConfig           `yaml:"Worker"`
	StorageProvider  StorageProviderConfig  `yaml:"StorageProvider"`

	TagCache         DockerCacheConfig `yaml:"TagCache"`
	DefaultPublisher string            `yaml:"DefaultPublisher"`

	ControlPlaneSettings RequesterControlPlaneConfig `yaml:"ControlPlaneSettings"`
}

type EvaluationBrokerConfig struct {
	EvalBrokerVisibilityTimeout    Duration `yaml:"EvalBrokerVisibilityTimeout"`
	EvalBrokerInitialRetryDelay    Duration `yaml:"EvalBrokerInitialRetryDelay"`
	EvalBrokerSubsequentRetryDelay Duration `yaml:"EvalBrokerSubsequentRetryDelay"`
	EvalBrokerMaxRetryCount        int      `yaml:"EvalBrokerMaxRetryCount"`
}

type WorkerConfig struct {
	WorkerCount                  int      `yaml:"WorkerCount"`
	WorkerEvalDequeueTimeout     Duration `yaml:"WorkerEvalDequeueTimeout"`
	WorkerEvalDequeueBaseBackoff Duration `yaml:"WorkerEvalDequeueBaseBackoff"`
	WorkerEvalDequeueMaxBackoff  Duration `yaml:"WorkerEvalDequeueMaxBackoff"`
}

type StorageProviderConfig struct {
	S3 S3StorageProviderConfig `yaml:"S3"`
}

type S3StorageProviderConfig struct {
	PreSignedURLDisabled   bool     `yaml:"PreSignedURLDisabled"`
	PreSignedURLExpiration Duration `yaml:"PreSignedURLExpiration"`
}

type JobDefaults struct {
	ExecutionTimeout Duration `yaml:"ExecutionTimeout"`
}

type RequesterControlPlaneConfig struct {
	// This setting is the time period after which a compute node is considered to be unresponsive.
	// If the compute node misses two of these frequencies, it will be marked as unknown.  The compute
	// node should have a frequency setting less than this one to ensure that it does not keep
	// switching between unknown and active too frequently.
	HeartbeatCheckFrequency Duration `yaml:"HeartbeatFrequency"`

	// This is the pubsub topic that the compute node will use to send heartbeats to the requester node.
	HeartbeatTopic string `yaml:"HeartbeatTopic"`

	// This is the time period after which a compute node is considered to be disconnected. If the compute
	// node does not deliver a heartbeat every `NodeDisconnectedAfter` then it is considered disconnected.
	NodeDisconnectedAfter Duration `yaml:"NodeDisconnectedAfter"`
}
