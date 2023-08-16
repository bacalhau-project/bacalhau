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

	LoggingMode logger.LogMode
	// Type is "compute", "requester" or both
	Type []string
	// Deprecated: TODO(forrest) remove.
	EstuaryAPIKey string
	// Local paths that are allowed to be mounted into jobs
	AllowListedLocalPaths []string
	// What feautres should not be enbaled even if installed
	DisabledFeatures FeatureConfig
	// Labels to apply to the node that can be used for node selection and filtering
	Labels map[string]string
}

type APIConfig struct {
	// Host is the hostname of an environment's public API servers.
	Host string
	// Port is the port that an environment serves the public API on.
	Port int
}

type Libp2pConfig struct {
	SwarmPort int
	// PeerConnect is the libp2p multiaddress to connect to.
	PeerConnect string
}

type IpfsConfig struct {
	// Connect is the multiaddress to connect to for IPFS.
	Connect string
	// Whether the in-process IPFS should automatically discover other IPFS nodes
	PrivateInternal bool
	// IPFS multiaddresses that the in-process IPFS should connect to
	SwarmAddresses []string
}

type FeatureConfig struct {
	Engines    []string
	Publishers []string
	Storages   []string
}
