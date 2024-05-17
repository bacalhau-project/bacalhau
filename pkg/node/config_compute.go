package node

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	pkgerrors "github.com/pkg/errors"
	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy/semantic"
	"github.com/bacalhau-project/bacalhau/pkg/compute/capacity"
	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

const (
	localPublishFolderPerm = 0755
)

// JobSelectionPolicy describe the rules for how a compute node selects an incoming job
type JobSelectionPolicy struct {
	// this describes if we should run a job based on
	// where the data is located - i.e. if the data is "local"
	// or if the data is "anywhere"
	Locality semantic.JobSelectionDataLocality `json:"locality"`
	// should we reject jobs that don't specify any data
	// the default is "accept"
	RejectStatelessJobs bool `json:"reject_stateless_jobs"`
	// should we accept jobs that specify networking
	// the default is "reject"
	AcceptNetworkedJobs bool `json:"accept_networked_jobs"`
	// external hooks that decide if we should take on the job or not
	// if either of these are given they will override the data locality settings
	ProbeHTTP string `json:"probe_http,omitempty"`
	ProbeExec string `json:"probe_exec,omitempty"`
}

type ComputeConfigParams struct {
	// Capacity config
	TotalResourceLimits          models.Resources
	JobResourceLimits            models.Resources
	DefaultJobResourceLimits     models.Resources
	PhysicalResourcesProvider    capacity.Provider
	IgnorePhysicalResourceLimits bool

	// Timeout config
	JobNegotiationTimeout      time.Duration
	MinJobExecutionTimeout     time.Duration
	MaxJobExecutionTimeout     time.Duration
	DefaultJobExecutionTimeout time.Duration

	JobExecutionTimeoutClientIDBypassList []string

	// Bid strategies config
	JobSelectionPolicy JobSelectionPolicy

	// logging running executions
	LogRunningExecutionsInterval time.Duration

	// How many messages to buffer in the log stream channel
	LogStreamBufferSize int

	FailureInjectionConfig model.FailureInjectionComputeConfig

	BidSemanticStrategy bidstrategy.SemanticBidStrategy
	BidResourceStrategy bidstrategy.ResourceBidStrategy

	ExecutionStore store.ExecutionStore

	LocalPublisher types.LocalPublisherConfig

	ControlPlaneSettings types.ComputeControlPlaneConfig
}

type ComputeConfig struct {
	// Capacity config
	TotalResourceLimits          models.Resources
	JobResourceLimits            models.Resources
	DefaultJobResourceLimits     models.Resources
	IgnorePhysicalResourceLimits bool

	// JobNegotiationTimeout default timeout value to hold a bid for a job
	JobNegotiationTimeout time.Duration
	// MinJobExecutionTimeout default value for the minimum execution timeout this compute node supports. Jobs with
	// lower timeout requirements will not be bid on.
	MinJobExecutionTimeout time.Duration
	// MaxJobExecutionTimeout default value for the maximum execution timeout this compute node supports. Jobs with
	// higher timeout requirements will not be bid on.
	MaxJobExecutionTimeout time.Duration
	// DefaultJobExecutionTimeout default value for the execution timeout this compute node will assign to jobs with
	// no timeout requirement defined.
	DefaultJobExecutionTimeout time.Duration

	// JobExecutionTimeoutClientIDBypassList is the list of clients that are allowed to bypass the job execution timeout
	// check.
	JobExecutionTimeoutClientIDBypassList []string

	// Bid strategies config
	JobSelectionPolicy JobSelectionPolicy

	// logging running executions
	LogRunningExecutionsInterval time.Duration

	// How many messages to buffer in the log stream channel
	LogStreamBufferSize int

	FailureInjectionConfig model.FailureInjectionComputeConfig

	BidSemanticStrategy bidstrategy.SemanticBidStrategy

	BidResourceStrategy bidstrategy.ResourceBidStrategy

	ExecutionStore store.ExecutionStore

	LocalPublisher types.LocalPublisherConfig

	ControlPlaneSettings types.ComputeControlPlaneConfig
}

func NewComputeConfigWithDefaults(storagePath string) (ComputeConfig, error) {
	return NewComputeConfigWith(storagePath, NewDefaultComputeParam(storagePath))
}

//nolint:funlen
func NewComputeConfigWith(storagePath string, params ComputeConfigParams) (ComputeConfig, error) {
	if storagePath == "" {
		storagePath = os.TempDir()
	}
	defaults := NewDefaultComputeParam(storagePath)
	if params.JobNegotiationTimeout == 0 {
		params.JobNegotiationTimeout = defaults.JobNegotiationTimeout
	}
	if params.MinJobExecutionTimeout == 0 {
		params.MinJobExecutionTimeout = defaults.MinJobExecutionTimeout
	}
	if params.MaxJobExecutionTimeout == 0 {
		params.MaxJobExecutionTimeout = defaults.MaxJobExecutionTimeout
	}
	if params.DefaultJobExecutionTimeout == 0 {
		params.DefaultJobExecutionTimeout = defaults.DefaultJobExecutionTimeout
	}
	if params.LogRunningExecutionsInterval == 0 {
		params.LogRunningExecutionsInterval = defaults.LogRunningExecutionsInterval
	}

	if params.LocalPublisher.Address == "" {
		params.LocalPublisher.Address = defaults.LocalPublisher.Address
	}
	if params.LocalPublisher.Directory == "" {
		params.LocalPublisher.Directory = defaults.LocalPublisher.Directory
		if err := os.MkdirAll(params.LocalPublisher.Directory, localPublishFolderPerm); err != nil {
			return ComputeConfig{}, pkgerrors.Wrap(err, "creating default local publisher directory")
		}
	}

	// Control plan settings defaults
	if params.ControlPlaneSettings.HeartbeatFrequency == 0 {
		params.ControlPlaneSettings.HeartbeatFrequency = defaults.ControlPlaneSettings.HeartbeatFrequency
	}
	if params.ControlPlaneSettings.InfoUpdateFrequency == 0 {
		params.ControlPlaneSettings.InfoUpdateFrequency = defaults.ControlPlaneSettings.InfoUpdateFrequency
	}
	if params.ControlPlaneSettings.ResourceUpdateFrequency == 0 {
		params.ControlPlaneSettings.ResourceUpdateFrequency = defaults.ControlPlaneSettings.ResourceUpdateFrequency
	}
	if params.ControlPlaneSettings.HeartbeatTopic == "" {
		params.ControlPlaneSettings.HeartbeatTopic = defaults.ControlPlaneSettings.HeartbeatTopic
	}

	// Get available physical resources in the host
	physicalResourcesProvider := params.PhysicalResourcesProvider
	if physicalResourcesProvider == nil {
		physicalResourcesProvider = defaults.PhysicalResourcesProvider
	}
	physicalResources, err := physicalResourcesProvider.GetAvailableCapacity(context.Background())
	if err != nil {
		return ComputeConfig{}, fmt.Errorf("getting compute resource capacity for config: %w", err)
	}
	// populate total resource limits with default values and physical resources if not set
	totalResourceLimits := params.TotalResourceLimits.
		Merge(defaults.TotalResourceLimits).
		Merge(physicalResources)

	// populate job resource limits with default values and total resource limits if not set
	jobResourceLimits := params.JobResourceLimits.
		Merge(defaults.JobResourceLimits).
		Merge(*totalResourceLimits)

	// populate default job resource limits with default values and job resource limits if not set
	defaultJobResourceLimits := params.DefaultJobResourceLimits.
		Merge(defaults.DefaultJobResourceLimits)

	config := ComputeConfig{
		TotalResourceLimits:          *totalResourceLimits,
		JobResourceLimits:            *jobResourceLimits,
		DefaultJobResourceLimits:     *defaultJobResourceLimits,
		IgnorePhysicalResourceLimits: params.IgnorePhysicalResourceLimits,

		JobNegotiationTimeout:      params.JobNegotiationTimeout,
		MinJobExecutionTimeout:     params.MinJobExecutionTimeout,
		MaxJobExecutionTimeout:     params.MaxJobExecutionTimeout,
		DefaultJobExecutionTimeout: params.DefaultJobExecutionTimeout,

		JobExecutionTimeoutClientIDBypassList: params.JobExecutionTimeoutClientIDBypassList,

		JobSelectionPolicy: params.JobSelectionPolicy,

		LogRunningExecutionsInterval: params.LogRunningExecutionsInterval,
		LogStreamBufferSize:          params.LogStreamBufferSize,
		FailureInjectionConfig:       params.FailureInjectionConfig,
		BidSemanticStrategy:          params.BidSemanticStrategy,
		BidResourceStrategy:          params.BidResourceStrategy,
		ExecutionStore:               params.ExecutionStore,
		LocalPublisher:               params.LocalPublisher,
		ControlPlaneSettings:         params.ControlPlaneSettings,
	}

	if err := validateConfig(config, physicalResources); err != nil {
		return ComputeConfig{}, fmt.Errorf("validating compute config: %w", err)
	}
	log.Debug().Msgf("Compute config: %+v", config)
	return config, nil
}

func validateConfig(config ComputeConfig, physicalResources models.Resources) error {
	var err error

	if !config.IgnorePhysicalResourceLimits && !config.TotalResourceLimits.LessThanEq(physicalResources) {
		err = errors.Join(err,
			fmt.Errorf("total resource limits %+v exceed physical resources %+v",
				config.TotalResourceLimits, physicalResources))
	}

	if !config.JobResourceLimits.LessThanEq(config.TotalResourceLimits) {
		err = errors.Join(err,
			fmt.Errorf("job resource limits %+v exceed total resource limits %+v",
				config.JobResourceLimits, config.TotalResourceLimits))
	}

	if !config.DefaultJobResourceLimits.LessThanEq(config.JobResourceLimits) {
		err = errors.Join(err,
			fmt.Errorf("default job resource limits %+v exceed job resource limits %+v",
				config.DefaultJobResourceLimits, config.JobResourceLimits))
	}

	return err
}
