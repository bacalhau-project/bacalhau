package node

import (
	"path"
	"runtime"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy/semantic"
	compute_system "github.com/bacalhau-project/bacalhau/pkg/compute/capacity/system"
	"github.com/bacalhau-project/bacalhau/pkg/config/cfgtypes"
	legacy_types "github.com/bacalhau-project/bacalhau/pkg/config_legacy/types"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/system"
)

func NewDefaultComputeParam(storagePath string) ComputeConfigParams {
	return ComputeConfigParams{
		PhysicalResourcesProvider: compute_system.NewPhysicalCapacityProvider(storagePath),
		DefaultJobResourceLimits: models.Resources{
			CPU:    0.1,               // 100m
			Memory: 100 * 1024 * 1024, // 100Mi
		},

		JobNegotiationTimeout:      3 * time.Minute,
		MinJobExecutionTimeout:     500 * time.Millisecond,
		MaxJobExecutionTimeout:     models.NoTimeout,
		DefaultJobExecutionTimeout: models.NoTimeout,

		LogRunningExecutionsInterval: 10 * time.Second,
		JobSelectionPolicy:           NewDefaultJobSelectionPolicy(),
		LocalPublisher: cfgtypes.LocalPublisher{
			Address:   "127.0.0.1",
			Port:      6001,
			Directory: path.Join(storagePath, "bacalhau-local-publisher"),
		},
		ControlPlaneSettings: legacy_types.ComputeControlPlaneConfig{
			InfoUpdateFrequency:     legacy_types.Duration(60 * time.Second), //nolint:gomnd
			ResourceUpdateFrequency: legacy_types.Duration(30 * time.Second), //nolint:gomnd
			HeartbeatFrequency:      legacy_types.Duration(15 * time.Second), //nolint:gomnd
			HeartbeatTopic:          "heartbeat",
		},
	}
}

var DefaultRequesterConfig = RequesterConfigParams{
	JobDefaults: cfgtypes.JobDefaults{
		Batch: cfgtypes.BatchJobDefaultsConfig{
			Task: cfgtypes.BatchTaskDefaultConfig{
				Timeouts: cfgtypes.TaskTimeoutConfig{
					TotalTimeout: cfgtypes.Duration(models.NoTimeout),
				},
			},
		},
		Ops: cfgtypes.BatchJobDefaultsConfig{
			Task: cfgtypes.BatchTaskDefaultConfig{
				Timeouts: cfgtypes.TaskTimeoutConfig{
					TotalTimeout: cfgtypes.Duration(models.NoTimeout),
				},
			},
		},
	},

	HousekeepingBackgroundTaskInterval: 30 * time.Second,
	HousekeepingTimeoutBuffer:          2 * time.Minute,
	NodeRankRandomnessRange:            5,
	OverAskForBidsFactor:               3,

	MinBacalhauVersion: models.BuildVersionInfo{
		Major: "1", Minor: "0", GitVersion: "v1.0.4",
	},

	EvalBrokerVisibilityTimeout:    60 * time.Second,
	EvalBrokerInitialRetryDelay:    1 * time.Second,
	EvalBrokerSubsequentRetryDelay: 30 * time.Second,
	EvalBrokerMaxRetryCount:        10,

	WorkerCount:                  runtime.NumCPU(),
	WorkerEvalDequeueTimeout:     5 * time.Second,
	WorkerEvalDequeueBaseBackoff: 1 * time.Second,
	WorkerEvalDequeueMaxBackoff:  30 * time.Second,

	NodeOverSubscriptionFactor: 1.5,

	S3PreSignedURLDisabled:   false,
	S3PreSignedURLExpiration: 30 * time.Minute,

	TranslationEnabled: false,

	ControlPlaneSettings: legacy_types.RequesterControlPlaneConfig{
		HeartbeatCheckFrequency: legacy_types.Duration(30 * time.Second), //nolint:gomnd
		HeartbeatTopic:          "heartbeat",
		NodeDisconnectedAfter:   legacy_types.Duration(30 * time.Second), //nolint:gomnd
	},

	NodeInfoStoreTTL:     10 * time.Minute,
	DefaultApprovalState: models.NodeMembership.APPROVED,
}

var TestRequesterConfig = RequesterConfigParams{
	JobDefaults: cfgtypes.JobDefaults{
		Batch: cfgtypes.BatchJobDefaultsConfig{
			Task: cfgtypes.BatchTaskDefaultConfig{
				Timeouts: cfgtypes.TaskTimeoutConfig{
					TotalTimeout: cfgtypes.Duration(30 * time.Second),
				},
			},
		},
		Ops: cfgtypes.BatchJobDefaultsConfig{
			Task: cfgtypes.BatchTaskDefaultConfig{
				Timeouts: cfgtypes.TaskTimeoutConfig{
					TotalTimeout: cfgtypes.Duration(30 * time.Second),
				},
			},
		},
	},

	HousekeepingBackgroundTaskInterval: 30 * time.Second,
	HousekeepingTimeoutBuffer:          100 * time.Millisecond,
	NodeRankRandomnessRange:            5,
	OverAskForBidsFactor:               3,

	MinBacalhauVersion: models.BuildVersionInfo{
		Major: "1", Minor: "0", GitVersion: "v1.0.4",
	},

	EvalBrokerVisibilityTimeout:    5 * time.Second,
	EvalBrokerInitialRetryDelay:    100 * time.Millisecond,
	EvalBrokerSubsequentRetryDelay: 100 * time.Millisecond,
	EvalBrokerMaxRetryCount:        3,

	WorkerCount:                  3,
	WorkerEvalDequeueTimeout:     200 * time.Millisecond,
	WorkerEvalDequeueBaseBackoff: 20 * time.Millisecond,
	WorkerEvalDequeueMaxBackoff:  200 * time.Millisecond,

	NodeOverSubscriptionFactor: 1.5,

	TranslationEnabled: false,

	S3PreSignedURLDisabled:   false,
	S3PreSignedURLExpiration: 30 * time.Minute,

	ControlPlaneSettings: legacy_types.RequesterControlPlaneConfig{
		HeartbeatCheckFrequency: legacy_types.Duration(30 * time.Second), //nolint:gomnd
		HeartbeatTopic:          "heartbeat",
		NodeDisconnectedAfter:   legacy_types.Duration(30 * time.Second), //nolint:gomnd
	},

	DefaultApprovalState: models.NodeMembership.APPROVED,
}

func getRequesterConfigParams() RequesterConfigParams {
	if system.GetEnvironment() == system.EnvironmentTest {
		return TestRequesterConfig
	}
	return DefaultRequesterConfig
}

func NewDefaultJobSelectionPolicy() JobSelectionPolicy {
	return JobSelectionPolicy{
		Locality: semantic.Anywhere,
	}
}
