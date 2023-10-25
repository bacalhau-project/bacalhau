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

var Production = types.BacalhauConfig{
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
			Host: "bootstrap.production.bacalhau.org",
			Port: 1234,
		},
		ServerAPI: types.APIConfig{
			Host: "0.0.0.0",
			Port: 1234,
			TLS:  types.TLSConfiguration{},
		},
		BootstrapAddresses: []string{
			"/ip4/35.245.115.191/tcp/1235/p2p/QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL",
			"/ip4/35.245.61.251/tcp/1235/p2p/QmXaXu9N5GNetatsvwnTfQqNtSeKAD6uCmarbh3LMRYAcF",
			"/ip4/35.245.251.239/tcp/1235/p2p/QmYgxZiySj3MRkwLSL4X2MF5F9f2PMhAE3LV49XkfNL1o3",
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
				"/ip4/35.245.115.191/tcp/4001/p2p/12D3KooWE4wfAknWtY9mQ4eAA8zrFGeZa7X2Kh4nBP2tZgDSt7Rh",
				"/ip4/35.245.115.191/udp/4001/quic/p2p/12D3KooWE4wfAknWtY9mQ4eAA8zrFGeZa7X2Kh4nBP2tZgDSt7Rh",
				"/ip4/35.245.61.251/tcp/4001/p2p/12D3KooWD8zeukHTMyuPtQBoUUPqtEnaA7NwFXWcVywUJtCVPske",
				"/ip4/35.245.61.251/udp/4001/quic/p2p/12D3KooWD8zeukHTMyuPtQBoUUPqtEnaA7NwFXWcVywUJtCVPske",
				"/ip4/35.245.251.239/tcp/4001/p2p/12D3KooWAg1YdehZxcZhetcgA6KP8TLGX6Fq4h9PUswnUWoStVNc",
				"/ip4/35.245.251.239/udp/4001/quic/p2p/12D3KooWAg1YdehZxcZhetcgA6KP8TLGX6Fq4h9PUswnUWoStVNc",
				"/ip4/34.150.153.87/tcp/4001/p2p/12D3KooWGE4R98vokeLsRVdTv8D6jhMnifo81mm7NMRV8WJPNVHb",
				"/ip4/34.150.153.87/udp/4001/quic/p2p/12D3KooWGE4R98vokeLsRVdTv8D6jhMnifo81mm7NMRV8WJPNVHb",
				"/ip4/34.91.247.176/tcp/4001/p2p/12D3KooWSNKPM5PBchoqn774bpQ4j4QbL3VoyX6mH6vTyWXqE3kH",
				"/ip4/34.91.247.176/udp/4001/quic/p2p/12D3KooWSNKPM5PBchoqn774bpQ4j4QbL3VoyX6mH6vTyWXqE3kH",
			},
		},
		Compute:   ProductionComputeConfig,
		Requester: ProductionRequesterConfig,
	},
}

var ProductionComputeConfig = types.ComputeConfig{
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
			CPU:    "500m",
			Memory: "1Gb",
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

var ProductionRequesterConfig = types.RequesterConfig{
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
