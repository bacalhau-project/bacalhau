package types

import "github.com/bacalhau-project/bacalhau/pkg/logger"

type NodeConfig struct {
	API    APIConfig
	Libp2p Libp2pConfig
	IPFS   IpfsConfig

	Compute   ComputeConfig
	Requester RequesterConfig

	// BootstrapAddresses is a list of bacalhau addresses for bootstrapping new local nodes.
	BootstrapAddresses []string

	DownloadURLRequestRetries int
	DownloadURLRequestTimeout Duration
	VolumeSizeRequestTimeout  Duration

	ExecutorPluginPath string

	ComputeStoragePath string

	LoggingMode           logger.LogMode
	Type                  []string
	EstuaryAPIKey         string
	AllowListedLocalPaths []string
	DisabledFeatures      FeatureConfig
	Labels                map[string]string
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

type FeatureConfig struct {
	Engines    []string
	Publishers []string
	Storages   []string
}
