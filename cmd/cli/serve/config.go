package serve

import "github.com/bacalhau-project/bacalhau/pkg/model"

var DefaultProductionBacalhauConfig = BacalhauConfig{
	Node: NodeConfig{
		Type:                  []string{"requester"},
		EstuaryAPIKey:         "",
		AllowListedLocalPaths: []string{},
		Labels:                map[string]string{},
		DisabledFeatures: FeatureConfig{
			Engines:    []string{},
			Publishers: []string{},
			Storages:   []string{},
		},
		API: APIConfig{
			Address: "bootstrap.production.bacalhau.org",
			Port:    1234,
		},
		Libp2p: Libp2pConfig{
			SwarmPort:   1235,
			PeerConnect: "none",
		},
		IPFS: IpfsConfig{
			Connect:         "",
			PrivateInternal: true,
			SwarmAddresses:  []string{},
		},
		Compute: ComputeConfig{
			ClientIDBypass:               []string{},
			IgnorePhysicalResourceLimits: false,
			Capacity: CapacityConfig{
				JobCPU:      "",
				JobMemory:   "",
				JobGPU:      "",
				TotalCPU:    "",
				TotalMemory: "",
				TotalGPU:    "",
			},
		},
		Requester: RequesterConfig{
			ExternalVerifierHook: "",
			JobSelectionPolicy: JobSelectionPolicyConfig{
				Locality:            model.Anywhere.String(),
				RejectStatelessJobs: false,
				AcceptNetworkedJobs: false,
				ProbeHTTP:           "",
				ProbeExec:           "",
			},
		},
	},
}

type BacalhauConfig struct {
	Node NodeConfig
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
	Locality            string
	RejectStatelessJobs bool
	AcceptNetworkedJobs bool
	ProbeHTTP           string
	ProbeExec           string
}
