package types

import (
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

type RequesterConfig struct {
	JobDefaults JobDefaults `yaml:"JobDefaults"`
	// URL where to send external verification requests to.
	ExternalVerifierHook string `yaml:"ExternalVerifierHook"`
	// How the node decides what jobs to run.
	JobSelectionPolicy model.JobSelectionPolicy `yaml:"JobSelectionPolicy"`
	JobStore           JobStoreConfig           `yaml:"JobStore"`

	NodeRanker NodeRankerConfig `yaml:"NodeRanker"`

	Housekeeping           HousekeepingConfig                    `yaml:"Housekeeping"`
	OverAskForBidsFactor   uint                                  `yaml:"OverAskForBidsFactor"`
	FailureInjectionConfig model.FailureInjectionRequesterConfig `yaml:"FailureInjectionConfig"`

	Translation TranslationConfig `yaml:"Translation"`

	EvaluationBroker EvaluationBrokerConfig `yaml:"EvaluationBroker"`
	Worker           WorkerConfig           `yaml:"Worker"`
	StorageProvider  StorageProviderConfig  `yaml:"StorageProvider"`

	TagCache DockerCacheConfig `yaml:"TagCache"`
}

type HousekeepingConfig struct {
	HousekeepingBackgroundTaskInterval Duration `yaml:"HousekeepingBackgroundTaskInterval"`
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

type NodeRankerConfig struct {
	// minimum version of compute nodes that the requester will accept and route jobs to
	MinBacalhauVersion      models.BuildVersionInfo `yaml:"MinBacalhauVersion"`
	NodeRankRandomnessRange int                     `yaml:"NodeRankRandomnessRange"`
}

type BuildVersionInfo struct {
	Major      string `json:"Major,omitempty" example:"0"`
	Minor      string `json:"Minor,omitempty" example:"3"`
	GitVersion string `json:"GitVersion" example:"v0.3.12"`
	GitCommit  string `json:"GitCommit" example:"d612b63108f2b5ce1ab2b9e02444eb1dac1d922d"`
	// TODO we need a special type for time in the config or need to change the codegen similar
	// to AsDuration
	BuildDate time.Time `json:"BuildDate" example:"2022-11-16T14:03:31Z"`
	GOOS      string    `json:"GOOS" example:"linux"`
	GOARCH    string    `json:"GOARCH" example:"amd64"`
}

type TranslationConfig struct {
	TranslationEnabled bool `yaml:"TranslationEnabled"`
}
