package types

import (
	"github.com/bacalhau-project/bacalhau/pkg/model"
)

type RequesterConfig struct {
	JobDefaults JobDefaults `yaml:"JobDefaults"`
	// URL where to send external verification requests to.
	ExternalVerifierHook string `yaml:"ExternalVerifierHook"`
	// How the node decides what jobs to run.
	// TODO(forrest) [refactor]: don't import models, define a new type
	JobSelectionPolicy model.JobSelectionPolicy `yaml:"JobSelectionPolicy"`
	JobStore           JobStoreConfig           `yaml:"JobStore"`

	Housekeeping            HousekeepingConfig `yaml:"Housekeeping"`
	NodeRankRandomnessRange int                `yaml:"NodeRankRandomnessRange"`
	OverAskForBidsFactor    uint               `yaml:"OverAskForBidsFactor"`
	// TODO(forrest) [refactor]: remove this field and use dep injection or mocks for testing.
	FailureInjectionConfig model.FailureInjectionRequesterConfig `yaml:"FailureInjectionConfig"`

	EvaluationBroker EvaluationBrokerConfig `yaml:"EvaluationBroker"`
	Worker           WorkerConfig           `yaml:"Worker"`
	StorageProvider  StorageProviderConfig  `yaml:"StorageProvider"`

	TagCache DockerCacheConfig `yaml:"TagCache"`

	ControlPlaneSettings RequesterControlPlaneConfig `yaml:"ControlPlaneSettings"`

	Translation    TranslationConfig    `yaml:"Translation"`
	NodeRanker     NodeRankerConfig     `yaml:"NodeRanker"`
	NodeMembership NodeMembershipConfig `yaml:"NodeMembership"`
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
	DefaultPublisher string   `yaml:"DefaultPublisher"`
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

type NodeRankerConfig struct {
	// minimum version of compute nodes that the requester will accept and route jobs to
	MinBacalhauVersion      BuildVersionInfo `yaml:"MinBacalhauVersion"`
	NodeRankRandomnessRange int              `yaml:"NodeRankRandomnessRange"`
}

type BuildVersionInfo struct {
	Major      string `json:"Major,omitempty" example:"0"`
	Minor      string `json:"Minor,omitempty" example:"3"`
	GitVersion string `json:"GitVersion" example:"v0.3.12"`
	GitCommit  string `json:"GitCommit" example:"d612b63108f2b5ce1ab2b9e02444eb1dac1d922d"`
}

type TranslationConfig struct {
	TranslationEnabled bool `yaml:"TranslationEnabled"`
}

type HousekeepingConfig struct {
	HousekeepingBackgroundTaskInterval Duration `yaml:"HousekeepingBackgroundTaskInterval"`
}

type NodeMembershipConfig struct {
	// when true nodes are automatically approved, else they are set to pending.
	AutoApproveNodes bool `yaml:"AutoApproveNodes"`
}
