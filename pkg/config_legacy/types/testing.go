//nolint:mnd  // Ignoring magic numbers in this configuration file, since it is easier to read that way
package types

import (
	"os"
	"runtime"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/authn"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/models"
)

var Testing = BacalhauConfig{
	Metrics: MetricsConfig{
		EventTracerPath: os.DevNull,
	},
	Update: UpdateConfig{
		SkipChecks: true,
	},
	Auth: AuthConfig{
		Methods: map[string]AuthenticatorConfig{
			"ClientKey": {
				Type: authn.MethodTypeChallenge,
			},
		},
	},
	Node: NodeConfig{
		NameProvider: "puuid",
		ClientAPI: APIConfig{
			Host: "test",
			Port: 9999,
		},
		ServerAPI: APIConfig{
			Host: "test",
			Port: 9999,
			TLS:  TLSConfiguration{},
		},
		Network: NetworkConfig{
			Port: 4222,
		},
		DownloadURLRequestTimeout: Duration(5 * time.Minute),
		VolumeSizeRequestTimeout:  Duration(2 * time.Minute),
		DownloadURLRequestRetries: 3,
		LoggingMode:               logger.LogModeDefault,
		Type:                      []string{"requester"},
		AllowListedLocalPaths:     []string{},
		Labels:                    map[string]string{},
		DisabledFeatures: FeatureConfig{
			Engines:    []string{},
			Publishers: []string{},
			Storages:   []string{},
		},
		IPFS: IpfsConfig{
			Connect: "",
		},
		Compute:   TestingComputeConfig,
		Requester: TestingRequesterConfig,
		WebUI: WebUIConfig{
			Enabled: false,
			Port:    8483,
		},
		StrictVersionMatch: false,
	},
}

var TestingComputeConfig = ComputeConfig{
	Capacity: CapacityConfig{
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
			CPU:    "100m",
			Memory: "100Mi",
			Disk:   "",
			GPU:    "",
		},
	},
	ExecutionStore: JobStoreConfig{
		Type: BoltDB,
		Path: "",
	},
	JobTimeouts: JobTimeoutConfig{
		JobExecutionTimeoutClientIDBypassList: []string{},
		JobNegotiationTimeout:                 Duration(3 * time.Minute),
		MinJobExecutionTimeout:                Duration(500 * time.Millisecond),
		MaxJobExecutionTimeout:                Duration(models.NoTimeout),
		DefaultJobExecutionTimeout:            Duration(10 * time.Minute),
	},
	JobSelection: models.JobSelectionPolicy{
		Locality:            models.Anywhere,
		RejectStatelessJobs: false,
		AcceptNetworkedJobs: false,
		ProbeHTTP:           "",
		ProbeExec:           "",
	},
	Logging: LoggingConfig{
		LogRunningExecutionsInterval: Duration(10 * time.Second),
	},
	ManifestCache: DockerCacheConfig{
		Size:      1000,
		Duration:  Duration(1 * time.Hour),
		Frequency: Duration(1 * time.Hour),
	},
	LogStreamConfig: LogStreamConfig{
		ChannelBufferSize: 10,
	},
	LocalPublisher: LocalPublisherConfig{
		Address: "private",
		Port:    6001,
	},
	ControlPlaneSettings: ComputeControlPlaneConfig{
		InfoUpdateFrequency:     Duration(60 * time.Second),
		ResourceUpdateFrequency: Duration(30 * time.Second),
		HeartbeatFrequency:      Duration(15 * time.Second),
		HeartbeatTopic:          "heartbeat",
	},
}

var TestingRequesterConfig = RequesterConfig{
	ExternalVerifierHook: "",
	JobSelectionPolicy: models.JobSelectionPolicy{
		Locality:            models.Anywhere,
		RejectStatelessJobs: false,
		AcceptNetworkedJobs: false,
		ProbeHTTP:           "",
		ProbeExec:           "",
	},
	JobStore: JobStoreConfig{
		Type: BoltDB,
		Path: "",
	},
	HousekeepingBackgroundTaskInterval: Duration(30 * time.Second),
	NodeRankRandomnessRange:            5,
	OverAskForBidsFactor:               3,
	FailureInjectionConfig: models.FailureInjectionConfig{
		IsBadActor: false,
	},
	EvaluationBroker: EvaluationBrokerConfig{
		EvalBrokerVisibilityTimeout:    Duration(60 * time.Second),
		EvalBrokerInitialRetryDelay:    Duration(1 * time.Second),
		EvalBrokerSubsequentRetryDelay: Duration(30 * time.Second),
		EvalBrokerMaxRetryCount:        10,
	},
	Worker: WorkerConfig{
		WorkerCount:                  runtime.NumCPU(),
		WorkerEvalDequeueTimeout:     Duration(5 * time.Second),
		WorkerEvalDequeueBaseBackoff: Duration(1 * time.Second),
		WorkerEvalDequeueMaxBackoff:  Duration(30 * time.Second),
	},
	Scheduler: SchedulerConfig{
		QueueBackoff:               Duration(5 * time.Second),
		NodeOverSubscriptionFactor: 1.5,
	},
	JobDefaults: JobDefaults{
		TotalTimeout: Duration(30 * time.Second),
	},
	StorageProvider: StorageProviderConfig{
		S3: S3StorageProviderConfig{
			PreSignedURLExpiration: Duration(30 * time.Minute),
		},
	},
	ControlPlaneSettings: RequesterControlPlaneConfig{
		HeartbeatCheckFrequency: Duration(30 * time.Second),
		HeartbeatTopic:          "heartbeat",
		NodeDisconnectedAfter:   Duration(30 * time.Second),
	},
	NodeInfoStoreTTL: Duration(10 * time.Minute),
}
