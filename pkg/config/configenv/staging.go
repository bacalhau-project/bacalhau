//nolint:gomnd
package configenv

import (
	"os"
	"runtime"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/models"
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
		Network: types.NetworkConfig{
			Type: models.NetworkTypeLibp2p,
			Port: 4222,
			Cluster: types.NetworkClusterConfig{
				Name: "global",
				Port: 6222,
			},
		},
		BootstrapAddresses: []string{
			"/ip4/34.85.228.65/tcp/1235/p2p/QmafZ9oCXCJZX9Wt1nhrGS9FVVq41qhcBRSNWCkVhz3Nvv",
			"/ip4/34.86.73.105/tcp/1235/p2p/QmVHCeiLzhFJPCyCj5S1RTAk1vBEvxd8r5A6E4HyJGQtbJ",
			"/ip4/34.150.138.100/tcp/1235/p2p/QmRr9qPTe4mU7aS9faKnWgvn1NtXt36FT8YUULRPCn2f3K",
		},
		DownloadURLRequestTimeout: types.Duration(300 * time.Second),
		VolumeSizeRequestTimeout:  types.Duration(2 * time.Minute),
		NodeInfoStoreTTL:          types.Duration(10 * time.Minute),
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
				"/ip4/34.85.228.65/tcp/4001/p2p/12D3KooWCWSTjjWh7SVoVv24W47z3T1Ly1tgnwZ56CCqCku5e4dS",
				"/ip4/34.85.228.65/udp/4001/quic/p2p/12D3KooWCWSTjjWh7SVoVv24W47z3T1Ly1tgnwZ56CCqCku5e4dS",
				"/ip4/34.86.73.105/tcp/4001/p2p/12D3KooWQuhW3LSpvhea25Zed47Z7fD5Cq2nw1xmapQ2tAUJ3q4F",
				"/ip4/34.86.73.105/udp/4001/quic/p2p/12D3KooWQuhW3LSpvhea25Zed47Z7fD5Cq2nw1xmapQ2tAUJ3q4F",
				"/ip4/34.150.138.100/tcp/4001/p2p/12D3KooWQm1T8EN8fMBz7rLviHxTGdRnohZ9nDPGbW4bfi78ckVT",
				"/ip4/34.150.138.100/udp/4001/quic/p2p/12D3KooWQm1T8EN8fMBz7rLviHxTGdRnohZ9nDPGbW4bfi78ckVT",
				"/ip4/35.245.247.85/tcp/4001/p2p/12D3KooWEztGEJtqtzy7th2d7cTw2iR4CQCPHFUYvj66rhh9Cf7h",
				"/ip4/35.245.247.85/udp/4001/quic/p2p/12D3KooWEztGEJtqtzy7th2d7cTw2iR4CQCPHFUYvj66rhh9Cf7h",
			},
			Profile:                "flatfs",
			SwarmListenAddresses:   []string{"/ip4/0.0.0.0/tcp/0", "/ip6/::1/tcp/0"},
			GatewayListenAddresses: []string{"/ip4/0.0.0.0/tcp/0", "/ip6/::1/tcp/0"},
			APIListenAddresses:     []string{"/ip4/0.0.0.0/tcp/0", "/ip6/::1/tcp/0"},
		},
		Compute:   StagingComputeConfig,
		Requester: StagingRequesterConfig,
		WebUI: types.WebUIConfig{
			Enabled: false,
			Port:    8483,
		},
	},
}

var StagingComputeConfig = types.ComputeConfig{
	Capacity: types.CapacityConfig{
		IgnorePhysicalResourceLimits: false,
		TotalResourceLimits: models.ResourcesConfig{
			CPU:    "",
			Memory: "",
			Disk:   "",
			GPU:    "",
		},
		JobResourceLimits: models.ResourcesConfig{
			CPU:    "",
			Memory: "",
			Disk:   "",
			GPU:    "",
		},
		DefaultJobResourceLimits: models.ResourcesConfig{
			CPU:    "500m",
			Memory: "1Gb",
			Disk:   "",
			GPU:    "",
		},
		QueueResourceLimits: models.ResourcesConfig{
			CPU:    "",
			Memory: "",
			Disk:   "",
			GPU:    "",
		},
	},
	ExecutionStore: types.JobStoreConfig{
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
	Queue: types.QueueConfig{},
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
	JobStore: types.JobStoreConfig{
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
	StorageProvider: types.StorageProviderConfig{
		S3: types.S3StorageProviderConfig{
			PreSignedURLExpiration: types.Duration(30 * time.Minute),
		},
	},
}
