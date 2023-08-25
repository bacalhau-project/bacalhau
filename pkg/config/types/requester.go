package types

import (
	"github.com/bacalhau-project/bacalhau/pkg/model"
)

type RequesterConfig struct {
	// URL where to send external verification requests to.
	ExternalVerifierHook string `yaml:"ExternalVerifierHook"`
	// How the node decides what jobs to run.
	JobSelectionPolicy model.JobSelectionPolicy `yaml:"JobSelectionPolicy"`
	JobStore           StorageConfig            `yaml:"JobStore"`

	HousekeepingBackgroundTaskInterval Duration                              `yaml:"HousekeepingBackgroundTaskInterval"`
	NodeRankRandomnessRange            int                                   `yaml:"NodeRankRandomnessRange"`
	OverAskForBidsFactor               uint                                  `yaml:"OverAskForBidsFactor"`
	FailureInjectionConfig             model.FailureInjectionRequesterConfig `yaml:"FailureInjectionConfig"`

	EvaluationBroker EvaluationBrokerConfig `yaml:"EvaluationBroker"`
	Worker           WorkerConfig           `yaml:"Worker"`
	Timeouts         TimeoutConfig          `yaml:"Timeouts"`
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

type TimeoutConfig struct {
	MinJobExecutionTimeout     Duration `yaml:"MinJobExecutionTimeout"`
	DefaultJobExecutionTimeout Duration `yaml:"DefaultJobExecutionTimeout"`
}
