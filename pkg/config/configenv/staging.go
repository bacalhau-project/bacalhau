//nolint:gomnd
package configenv

import (
	"os"
	"runtime"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/model"
)

var Staging = types.BacalhauConfig{
	Metrics: types.MetricsConfig{
		Libp2pTracerPath: os.DevNull,
		EventTracerPath:  os.DevNull,
	},
	Node: types.NodeConfig{
		ClientAPI: types.APIConfig{
			Host: "bootstrap.staging.bacalhau.org",
			Port: 1234,
		},
		ServerAPI: types.APIConfig{
			Host: "0.0.0.0",
			Port: 1234,
			TLS:  types.TLSConfiguration{},
		},
		BootstrapAddresses: []string{
			"/ip4/34.85.197.247/tcp/1235/p2p/QmSnwDbJYYrFsDGfqYjWjktUMKePTBEMDVauLPKbWKb8rr",
			"/ip4/35.199.63.137/tcp/1235/p2p/Qmf2U97oNUtyH8tSJ8yHqWmLU32PX2bNxhhsdETd7hFYMG",
			"/ip4/35.188.227.44/tcp/1235/p2p/QmTMyorSBpERJT5hukKi3p24zC1qHxNKZPxiBaveMSjXZ8",
		},
		DownloadURLRequestTimeout: types.Duration(300 * time.Second),
		VolumeSizeRequestTimeout:  types.Duration(2 * time.Minute),
		DownloadURLRequestRetries: 3,
		LoggingMode:               logger.LogModeDefault,
		Type:                      []string{"requester"},
		EstuaryAPIKey:             "",
		AllowListedLocalPaths:     []string{},
		Labels:                    map[string]string{},
		DisabledFeatures: types.FeatureConfig{
			Engines:    []string{},
			Publishers: []string{},
			Storages:   []string{},
		},
		Libp2p: types.Libp2pConfig{
			SwarmPort:   1235,
			PeerConnect: "none",
		},
		IPFS: types.IpfsConfig{
			Connect:         "",
			PrivateInternal: true,
			SwarmAddresses: []string{
				"/ip4/34.85.197.247/tcp/1235/p2p/QmSnwDbJYYrFsDGfqYjWjktUMKePTBEMDVauLPKbWKb8rr",
				"/ip4/35.199.63.137/tcp/1235/p2p/Qmf2U97oNUtyH8tSJ8yHqWmLU32PX2bNxhhsdETd7hFYMG",
				"/ip4/35.188.227.44/tcp/1235/p2p/QmTMyorSBpERJT5hukKi3p24zC1qHxNKZPxiBaveMSjXZ8",
			},
		},
		Compute:   StagingComputeConfig,
		Requester: StagingRequesterConfig,
	},
}

var StagingComputeConfig = types.ComputeConfig{
	Capacity: types.CapacityConfig{
		IgnorePhysicalResourceLimits: false,
		TotalResourceLimits: model.ResourceUsageConfig{
			CPU:    "",
			Memory: "",
			Disk:   "",
			GPU:    "",
		},
		JobResourceLimits: model.ResourceUsageConfig{
			CPU:    "",
			Memory: "",
			Disk:   "",
			GPU:    "",
		},
		DefaultJobResourceLimits: model.ResourceUsageConfig{
			CPU:    "100m",
			Memory: "100Mi",
			Disk:   "",
			GPU:    "",
		},
		QueueResourceLimits: model.ResourceUsageConfig{
			CPU:    "",
			Memory: "",
			Disk:   "",
			GPU:    "",
		},
	},
	ExecutionStore: types.StorageConfig{
		Type: types.BoltDB,
		Path: "",
	},
	JobTimeouts: types.JobTimeoutConfig{
		JobExecutionTimeoutClientIDBypassList: []string{},
		JobNegotiationTimeout:                 types.Duration(3 * time.Minute),
		MinJobExecutionTimeout:                types.Duration(500 * time.Millisecond),
		MaxJobExecutionTimeout:                types.Duration(model.NoJobTimeout),
		DefaultJobExecutionTimeout:            types.Duration(10 * time.Minute),
	},
	JobSelection: model.JobSelectionPolicy{
		Locality:            model.Anywhere,
		RejectStatelessJobs: false,
		AcceptNetworkedJobs: false,
		ProbeHTTP:           "",
		ProbeExec:           "",
	},
	Queue: types.QueueConfig{
		ExecutorBufferBackoffDuration: types.Duration(50 * time.Millisecond),
	},
	Logging: types.LoggingConfig{
		LogRunningExecutionsInterval: types.Duration(10 * time.Second),
	},
}

var StagingRequesterConfig = types.RequesterConfig{
	ExternalVerifierHook: "",
	JobSelectionPolicy: model.JobSelectionPolicy{
		Locality:            model.Anywhere,
		RejectStatelessJobs: false,
		AcceptNetworkedJobs: false,
		ProbeHTTP:           "",
		ProbeExec:           "",
	},
	JobStore: types.StorageConfig{
		Type: types.BoltDB,
		Path: "",
	},
	HousekeepingBackgroundTaskInterval: types.Duration(30 * time.Second),
	NodeRankRandomnessRange:            5,
	OverAskForBidsFactor:               3,
	FailureInjectionConfig: model.FailureInjectionRequesterConfig{
		IsBadActor: false,
	},
	EvaluationBroker: types.EvaluationBrokerConfig{
		EvalBrokerVisibilityTimeout:    types.Duration(60 * time.Second),
		EvalBrokerInitialRetryDelay:    types.Duration(1 * time.Second),
		EvalBrokerSubsequentRetryDelay: types.Duration(30 * time.Second),
		EvalBrokerMaxRetryCount:        10,
	},
	Worker: types.WorkerConfig{
		WorkerCount:                  runtime.NumCPU(),
		WorkerEvalDequeueTimeout:     types.Duration(5 * time.Second),
		WorkerEvalDequeueBaseBackoff: types.Duration(1 * time.Second),
		WorkerEvalDequeueMaxBackoff:  types.Duration(30 * time.Second),
	},
	JobDefaults: types.JobDefaults{
		ExecutionTimeout: types.Duration(30 * time.Minute),
	},
}
