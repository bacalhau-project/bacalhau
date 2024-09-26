package types

import (
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

type RequesterConfig struct {
	JobDefaults JobDefaults `yaml:"JobDefaults"`
	// URL where to send external verification requests to.
	ExternalVerifierHook string `yaml:"ExternalVerifierHook"`
	// How the node decides what jobs to run.
	JobSelectionPolicy models.JobSelectionPolicy `yaml:"JobSelectionPolicy"`
	JobStore           JobStoreConfig            `yaml:"JobStore"`

	HousekeepingBackgroundTaskInterval Duration                               `yaml:"HousekeepingBackgroundTaskInterval" swaggertype:"primitive,integer"`
	NodeRankRandomnessRange            int                                    `yaml:"NodeRankRandomnessRange"`
	OverAskForBidsFactor               uint                                   `yaml:"OverAskForBidsFactor"`
	FailureInjectionConfig             models.FailureInjectionRequesterConfig `yaml:"FailureInjectionConfig"`

	TranslationEnabled bool `yaml:"TranslationEnabled"`

	EvaluationBroker EvaluationBrokerConfig `yaml:"EvaluationBroker"`
	Worker           WorkerConfig           `yaml:"Worker"`
	Scheduler        SchedulerConfig        `yaml:"Scheduler"`
	StorageProvider  StorageProviderConfig  `yaml:"StorageProvider"`

	TagCache         DockerCacheConfig `yaml:"TagCache"`
	DefaultPublisher string            `yaml:"DefaultPublisher"`

	ControlPlaneSettings RequesterControlPlaneConfig `yaml:"ControlPlaneSettings"`
	NodeInfoStoreTTL     Duration                    `yaml:"NodeInfoStoreTTL" swaggertype:"primitive,integer"`

	// ManualNodeApproval is a flag that determines if nodes should be manually approved or not.
	// By default, nodes are auto-approved to simplify upgrades, by setting this property to
	// true, nodes will need to be manually approved before they are included in node selection.
	ManualNodeApproval bool `yaml:"ManualNodeApproval"`
}

type EvaluationBrokerConfig struct {
	EvalBrokerVisibilityTimeout    Duration `yaml:"EvalBrokerVisibilityTimeout" swaggertype:"primitive,integer"`
	EvalBrokerInitialRetryDelay    Duration `yaml:"EvalBrokerInitialRetryDelay" swaggertype:"primitive,integer"`
	EvalBrokerSubsequentRetryDelay Duration `yaml:"EvalBrokerSubsequentRetryDelay" swaggertype:"primitive,integer"`
	EvalBrokerMaxRetryCount        int      `yaml:"EvalBrokerMaxRetryCount"`
}

type WorkerConfig struct {
	WorkerCount                  int      `yaml:"WorkerCount"`
	WorkerEvalDequeueTimeout     Duration `yaml:"WorkerEvalDequeueTimeout" swaggertype:"primitive,integer"`
	WorkerEvalDequeueBaseBackoff Duration `yaml:"WorkerEvalDequeueBaseBackoff" swaggertype:"primitive,integer"`
	WorkerEvalDequeueMaxBackoff  Duration `yaml:"WorkerEvalDequeueMaxBackoff" swaggertype:"primitive,integer"`
}

type SchedulerConfig struct {
	QueueBackoff               Duration `yaml:"QueueBackoff" swaggertype:"primitive,integer"`
	NodeOverSubscriptionFactor float64  `yaml:"NodeOverSubscriptionFactor"`
}

type StorageProviderConfig struct {
	S3 S3StorageProviderConfig `yaml:"S3"`
}

type S3StorageProviderConfig struct {
	PreSignedURLDisabled   bool     `yaml:"PreSignedURLDisabled"`
	PreSignedURLExpiration Duration `yaml:"PreSignedURLExpiration" swaggertype:"primitive,integer"`
}

type JobDefaults struct {
	TotalTimeout     Duration `yaml:"TotalTimeout" swaggertype:"primitive,integer"`
	ExecutionTimeout Duration `yaml:"ExecutionTimeout" swaggertype:"primitive,integer"`
	QueueTimeout     Duration `yaml:"QueueTimeout" swaggertype:"primitive,integer"`
}

type RequesterControlPlaneConfig struct {
	// This setting is the time period after which a compute node is considered to be unresponsive.
	// If the compute node misses two of these frequencies, it will be marked as unknown.  The compute
	// node should have a frequency setting less than this one to ensure that it does not keep
	// switching between unknown and active too frequently.
	HeartbeatCheckFrequency Duration `yaml:"HeartbeatFrequency" swaggertype:"primitive,integer"`

	// This is the pubsub topic that the compute node will use to send heartbeats to the requester node.
	HeartbeatTopic string `yaml:"HeartbeatTopic"`

	// This is the time period after which a compute node is considered to be disconnected. If the compute
	// node does not deliver a heartbeat every `NodeDisconnectedAfter` then it is considered disconnected.
	NodeDisconnectedAfter Duration `yaml:"NodeDisconnectedAfter" swaggertype:"primitive,integer"`
}
