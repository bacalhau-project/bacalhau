package config_v2

import (
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/model"
)

var Default BacalhauConfig

//go:generate go run gen_paths/generate.go
//go:generate go run gen_viper/generate.go
type BacalhauConfig struct {
	Node    NodeConfig
	User    UserConfig
	Metrics MetricsConfig
}

type MetricsConfig struct {
	Libp2pTracerPath string
	EventTracerPath  string
}

type UserConfig struct {
	UserKeyPath   string
	Libp2pKeyPath string
}

type FeatureConfig struct {
	Engines    []string
	Publishers []string
	Storages   []string
}

type NodeConfig struct {
	// BootstrapAddresses is a list of bacalhau addresses for bootstrapping new local nodes.
	BootstrapAddresses []string

	DownloadURLRequestRetries int
	DownloadURLRequestTimeout time.Duration
	VolumeSizeRequestTimeout  time.Duration

	ExecutorPluginPath string

	ComputeStoragePath string

	LoggingMode           logger.LogMode
	Type                  []string
	EstuaryAPIKey         string
	AllowListedLocalPaths []string
	DisabledFeatures      FeatureConfig
	Labels                map[string]string

	API       APIConfig
	Libp2p    Libp2pConfig
	IPFS      IpfsConfig
	Compute   ComputeConfig
	Requester RequesterConfig
}

type APIConfig struct {
	// Host is the hostname of an environment's public API servers.
	Host string
	// Port is the port that an environment serves the public API on.
	Port int
}

type Libp2pConfig struct {
	SwarmPort   int
	PeerConnect string
}

type IpfsConfig struct {
	Connect         string
	PrivateInternal bool
	SwarmAddresses  []string
}

type ComputeConfig struct {
	ClientIDBypass               []string
	IgnorePhysicalResourceLimits bool
	Capacity                     CapacityConfig
	ExecutionStore               StorageConfig
}

type CapacityConfig struct {
	JobCPU      string
	JobMemory   string
	JobGPU      string
	TotalCPU    string
	TotalMemory string
	TotalGPU    string
}

type RequesterConfig struct {
	ExternalVerifierHook string
	JobSelectionPolicy   JobSelectionPolicyConfig
	JobStore             StorageConfig
}

type JobSelectionPolicyConfig struct {
	Locality            model.JobSelectionDataLocality
	RejectStatelessJobs bool
	AcceptNetworkedJobs bool
	ProbeHTTP           string
	ProbeExec           string
}
