package types

import (
	"github.com/bacalhau-project/bacalhau/pkg/logger"
)

type NodeConfig struct {
	Name         string       `yaml:"Name"`
	NameProvider string       `yaml:"NameProvider"`
	ClientAPI    APIConfig    `yaml:"ClientAPI"`
	ServerAPI    APIConfig    `yaml:"ServerAPI"`
	Libp2p       Libp2pConfig `yaml:"Libp2P"`
	IPFS         IpfsConfig   `yaml:"IPFS"`

	Compute   ComputeConfig   `yaml:"Compute"`
	Requester RequesterConfig `yaml:"Requester"`

	// BootstrapAddresses is a list of bacalhau addresses for bootstrapping new local nodes.
	BootstrapAddresses []string `yaml:"BootstrapAddresses"`

	DownloadURLRequestRetries int      `yaml:"DownloadURLRequestRetries"`
	DownloadURLRequestTimeout Duration `yaml:"DownloadURLRequestTimeout"`
	VolumeSizeRequestTimeout  Duration `yaml:"VolumeSizeRequestTimeout"`
	NodeInfoStoreTTL          Duration `yaml:"NodeInfoStoreTTL"`

	ExecutorPluginPath string `yaml:"ExecutorPluginPath"`

	ComputeStoragePath string `yaml:"ComputeStoragePath"`

	LoggingMode logger.LogMode `yaml:"LoggingMode"`
	// Type is "compute", "requester" or both
	Type []string `yaml:"Type"`
	// Local paths that are allowed to be mounted into jobs
	AllowListedLocalPaths []string `yaml:"AllowListedLocalPaths"`
	// What features should not be enabled even if installed
	DisabledFeatures FeatureConfig `yaml:"DisabledFeatures"`
	// Labels to apply to the node that can be used for node selection and filtering
	Labels map[string]string `yaml:"Labels"`

	// Configuration for the web UI
	WebUI WebUIConfig `yaml:"WebUI"`

	Network NetworkConfig `yaml:"Network"`

	StrictVersionMatch bool `yaml:"StrictVersionMatch"`
}

type APIConfig struct {
	// Host is the hostname of an environment's public API servers.
	Host string `yaml:"Host"`
	// Port is the port that an environment serves the public API on.
	Port int `yaml:"Port"`

	// ClientTLS specifies tls options for the client connecting to the
	// API.
	ClientTLS ClientTLSConfig `yaml:"ClientTLS"`

	// TLS returns information about how TLS is configured for the public server.
	// This is only used in APIConfig for NodeConfig.ServerAPI
	TLS TLSConfiguration `yaml:"TLS"`
}

type ClientTLSConfig struct {
	// Used for NodeConfig.ClientAPI, instructs the client to connect over
	// TLS.  Auto enabled if Insecure or CACert are specified.
	UseTLS bool `yaml:"UseTLS"`

	// Used for NodeConfig.ClientAPI, specifies the location of a ca certificate
	// file (primarily for self-signed server certs). Will use HTTPS for requests.
	CACert string `yaml:"CACert"`

	// Used for NodeConfig.ClientAPI, and when true instructs the client to use
	// HTTPS, but not to attempt to verify the certificate.
	Insecure bool `yaml:"Insecure"`
}

type WebUIConfig struct {
	Enabled bool `yaml:"Enabled"`
	Port    int  `yaml:"Port"`
}

type TLSConfiguration struct {
	// AutoCert specifies a hostname for a certificate to be obtained via ACME.
	// This is only used by the server, and only by the requester node when it
	// has a publicly resolvable domain name.
	AutoCert string `yaml:"AutoCert"`

	// AutoCertCachePath specifies the directory where the autocert process
	// will cache certificates to avoid rate limits.
	AutoCertCachePath string `yaml:"AutoCertCachePath"`

	// ServerCertificate specifies the location of a TLS certificate to be used
	// by the requester to serve TLS requests
	ServerCertificate string `yaml:"ServerCertificate"`

	// ServerKey is the TLS server key to match the certificate to allow the
	// requester to server TLS.
	ServerKey string `yaml:"ServerKey"`

	// SelfSignedCert will auto-generate a self-signed certificate for the
	// requester node if TLS certificates have not been provided.
	SelfSigned bool `yaml:"SelfSigned"`
}

type Libp2pConfig struct {
	SwarmPort int `yaml:"SwarmPort"`
	// PeerConnect is the libp2p multiaddress to connect to.
	PeerConnect string `yaml:"PeerConnect"`
}

type IpfsConfig struct {
	// Connect is the multiaddress to connect to for IPFS.
	Connect string `yaml:"Connect"`
}

type FeatureConfig struct {
	Engines    []string `yaml:"Engines"`
	Publishers []string `yaml:"Publishers"`
	Storages   []string `yaml:"Storages"`
}

type DockerCacheConfig struct {
	Size      uint64   `yaml:"Size"`
	Duration  Duration `yaml:"Duration"`
	Frequency Duration `yaml:"Frequency"`
}

type NetworkConfig struct {
	Type              string               `yaml:"Type"`
	Port              int                  `yaml:"Port"`
	AdvertisedAddress string               `yaml:"AdvertisedAddress"`
	AuthSecret        string               `yaml:"AuthSecret"`
	Orchestrators     []string             `yaml:"Orchestrators"`
	StoreDir          string               `yaml:"StoreDir"`
	Cluster           NetworkClusterConfig `yaml:"Cluster"`
}

type NetworkClusterConfig struct {
	Name              string   `yaml:"Name"`
	Port              int      `yaml:"Port"`
	AdvertisedAddress string   `yaml:"AdvertisedAddress"`
	Peers             []string `yaml:"Peers"`
}
