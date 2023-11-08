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

var Development = types.BacalhauConfig{
	Metrics: types.MetricsConfig{
		Libp2pTracerPath: os.DevNull,
		EventTracerPath:  os.DevNull,
	},
	Update: types.UpdateConfig{
		SkipChecks: true,
	},
	Node: types.NodeConfig{
		ClientAPI: types.APIConfig{
			Host: "bootstrap.development.bacalhau.org",
			Port: 1234,
		},
		ServerAPI: types.APIConfig{
			Host: "0.0.0.0",
			Port: 1234,
			TLS:  types.TLSConfiguration{},
		},
		BootstrapAddresses: []string{
			"/ip4/34.88.135.65/tcp/1235/p2p/QmfRDVYnEcPassyJFGQw8Wt4t9QuA843uuKPVNEVNm4Smo",
			"/ip4/35.228.112.50/tcp/1235/p2p/QmQM1yRXyKGAfFtYpPSy5grHSief3fic6YjLEWQYpmiGTM",
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
				"/ip4/34.88.135.65/tcp/4001/p2p/12D3KooWMYUaxS32AucbSN4Y9ae6CPfHcZiLxteAQpbjTBHexG6N",
				"/ip4/34.88.135.65/udp/4001/quic/p2p/12D3KooWMYUaxS32AucbSN4Y9ae6CPfHcZiLxteAQpbjTBHexG6N",
				"/ip4/35.228.112.50/tcp/4001/p2p/12D3KooWJE4HiVBbyUp2x3B98xKukKbN76zvU4FN8Uvc1NWZYzHS",
				"/ip4/35.228.112.50/udp/4001/quic/p2p/12D3KooWJE4HiVBbyUp2x3B98xKukKbN76zvU4FN8Uvc1NWZYzHS",
			},
		},
		Compute:   DevelopmentComputeConfig,
		Requester: DevelopmentRequesterConfig,
	},
}

var DevelopmentComputeConfig = types.ComputeConfig{
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

var DevelopmentRequesterConfig = types.RequesterConfig{
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
