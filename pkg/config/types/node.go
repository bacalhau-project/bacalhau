package types

import "github.com/bacalhau-project/bacalhau/pkg/logger"

type NodeConfig struct {
	ClientAPI APIConfig    `yaml:"ClientAPI"`
	ServerAPI APIConfig    `yaml:"ServerAPI"`
	Libp2p    Libp2pConfig `yaml:"Libp2P"`
	IPFS      IpfsConfig   `yaml:"IPFS"`

	Compute   ComputeConfig   `yaml:"Compute"`
	Requester RequesterConfig `yaml:"Requester"`

	// BootstrapAddresses is a list of bacalhau addresses for bootstrapping new local nodes.
	BootstrapAddresses []string `yaml:"BootstrapAddresses"`

	DownloadURLRequestRetries int      `yaml:"DownloadURLRequestRetries"`
	DownloadURLRequestTimeout Duration `yaml:"DownloadURLRequestTimeout"`
	VolumeSizeRequestTimeout  Duration `yaml:"VolumeSizeRequestTimeout"`

	ExecutorPluginPath string `yaml:"ExecutorPluginPath"`

	ComputeStoragePath string `yaml:"ComputeStoragePath"`

	LoggingMode logger.LogMode `yaml:"LoggingMode"`
	// Type is "compute", "requester" or both
	Type []string `yaml:"Type"`
	// Deprecated: TODO(forrest) remove.
	EstuaryAPIKey string `yaml:"EstuaryAPIKey"`
	// Local paths that are allowed to be mounted into jobs
	AllowListedLocalPaths []string `yaml:"AllowListedLocalPaths"`
	// What feautres should not be enbaled even if installed
	DisabledFeatures FeatureConfig `yaml:"DisabledFeatures"`
	// Labels to apply to the node that can be used for node selection and filtering
	Labels map[string]string `yaml:"Labels"`
}

type APIConfig struct {
	// Host is the hostname of an environment's public API servers.
	Host string `yaml:"Host"`
	// Port is the port that an environment serves the public API on.
	Port int `yaml:"Port"`
	// TLS returns information about how TLS is configured for the public server
	TLS TLSConfiguration `yaml:"TLS"`
}

type TLSConfiguration struct {
	// AutoCert specifies a hostname for a certificate to be obtained via ACME.
	// This is only used by the server, and only by the requester node when it
	// has a publicly resolvable domain name.
	AutoCert string `yaml:"AutoCert"`

	// AutoCertCachePath specifies the directory where the autocert process
	// will cache certificates to avoid rate limits.
	AutoCertCachePath string `yaml:"AutoCertCachePath"`
}

type Libp2pConfig struct {
	SwarmPort int `yaml:"SwarmPort"`
	// PeerConnect is the libp2p multiaddress to connect to.
	PeerConnect string `yaml:"PeerConnect"`
}

type IpfsConfig struct {
	// Connect is the multiaddress to connect to for IPFS.
	Connect string `yaml:"Connect"`
	// Whether the in-process IPFS should automatically discover other IPFS nodes
	PrivateInternal bool `yaml:"PrivateInternal"`
	// IPFS multiaddresses that the in-process IPFS should connect to
	SwarmAddresses []string `yaml:"SwarmAddresses"`
	// Path of the IPFS repo
	ServePath string `yaml:"ServePath"`
}

type FeatureConfig struct {
	Engines    []string `yaml:"Engines"`
	Publishers []string `yaml:"Publishers"`
	Storages   []string `yaml:"Storages"`
}
