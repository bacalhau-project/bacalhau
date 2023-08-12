package types

import (
	"github.com/bacalhau-project/bacalhau/pkg/model"
)

type RequesterConfig struct {
	ExternalVerifierHook string
	JobSelectionPolicy   model.JobSelectionPolicy
	JobStore             StorageConfig

	HousekeepingBackgroundTaskInterval Duration
	NodeRankRandomnessRange            int
	OverAskForBidsFactor               uint
	FailureInjectionConfig             model.FailureInjectionRequesterConfig

	EvaluationBroker EvaluationBrokerConfig
	Worker           WorkerConfig
	Timeouts         TimeoutConfig
}

type EvaluationBrokerConfig struct {
	EvalBrokerVisibilityTimeout    Duration
	EvalBrokerInitialRetryDelay    Duration
	EvalBrokerSubsequentRetryDelay Duration
	EvalBrokerMaxRetryCount        int
}

type WorkerConfig struct {
	WorkerCount                  int
	WorkerEvalDequeueTimeout     Duration
	WorkerEvalDequeueBaseBackoff Duration
	WorkerEvalDequeueMaxBackoff  Duration
}

type TimeoutConfig struct {
	MinJobExecutionTimeout     Duration
	DefaultJobExecutionTimeout Duration
}
