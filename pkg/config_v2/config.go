package config_v2

import (
	"github.com/bacalhau-project/bacalhau/pkg/model"
)

var Default BacalhauConfig

//go:generate go run gen_paths/generate.go
//go:generate go run gen_viper/generate.go
type BacalhauConfig struct {
	Environment EnvironmentConfig
	Node        NodeConfig
}

type EnvironmentConfig struct {
	// APIHost is the hostname of an environment's public API servers.
	APIHost string
	// APIPort is the port that an environment serves the public API on.
	APIPort uint16
	// BootstrapAddresses is a list of bacalhau addresses for bootstrapping new local nodes.
	BootstrapAddresses []string
	// IPFSSwarmAddresses lists the swarm addresses of an environment's IPFS
	// nodes, for bootstrapping new local nodes.
	IPFSSwarmAddresses []string
}

type FeatureConfig struct {
	Engines    []string
	Publishers []string
	Storages   []string
}

type NodeConfig struct {
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
	Address string
	Port    int
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
}

type JobSelectionPolicyConfig struct {
	Locality            model.JobSelectionDataLocality
	RejectStatelessJobs bool
	AcceptNetworkedJobs bool
	ProbeHTTP           string
	ProbeExec           string
}
