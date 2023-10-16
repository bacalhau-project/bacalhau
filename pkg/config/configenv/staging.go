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
	Update: types.UpdateConfig{
		SkipChecks:     false,
		CheckFrequency: types.Duration(24 * time.Hour),
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
			"/ip4/34.85.197.247/tcp/1235/p2p/QmcWJnVXJ82DKJq8ED79LADR4ZBTnwgTK7yn6JQbNVMbbC",
			"/ip4/35.245.163.45/tcp/1235/p2p/QmXRdLruWyETS2Z8XFrXxBFYXctfjT8T9mZWyuqwUm6rQk",
			"/ip4/35.199.63.137/tcp/1235/p2p/QmVXwmdZUHsa88UHRKwQvvapguAXgHxd2uvVEpHrhjchAH",
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
			// Swarm addresses of the IPFS nodes. Find these by running: `env IPFS_PATH=/data/ipfs ipfs id`.
			SwarmAddresses: []string{
				"/ip4/34.85.197.247/tcp/4001/p2p/12D3KooWDaveKrxPZSCrHY2hjsk4pcWVSCE3eUKFotmKWL8nJDZi",
				"/ip4/34.85.197.247/udp/4001/quic/p2p/12D3KooWDaveKrxPZSCrHY2hjsk4pcWVSCE3eUKFotmKWL8nJDZi",
				"/ip4/35.245.163.45/tcp/4001/p2p/12D3KooWDzhR9PAsCNc2xY7v6NQJ4oKEmhvpwaxKCS3Jbxw36G3C",
				"/ip4/35.245.163.45/udp/4001/quic/p2p/12D3KooWDzhR9PAsCNc2xY7v6NQJ4oKEmhvpwaxKCS3Jbxw36G3C",
				"/ip4/35.199.63.137/tcp/4001/p2p/12D3KooWKFN8bf1yK1n2zXZMUp2NpTcCLn9szcpYiwrrg9YbrKap",
				"/ip4/35.199.63.137/udp/4001/quic/p2p/12D3KooWKFN8bf1yK1n2zXZMUp2NpTcCLn9szcpYiwrrg9YbrKap",
				"/ip4/35.188.227.44/tcp/4001/p2p/12D3KooWGzqdeV1oM7voQghTtnwSQmmfadPswTweDTtya6qMJC3j",
				"/ip4/35.188.227.44/udp/4001/quic/p2p/12D3KooWGzqdeV1oM7voQghTtnwSQmmfadPswTweDTtya6qMJC3j",
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
