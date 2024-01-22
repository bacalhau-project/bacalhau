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
