//go:generate go run gen/gen.go > viper_defaults.go

package config_v2

import (
	"github.com/bacalhau-project/bacalhau/pkg/model"
)

func init() {
	Default = Production()
}

func Production() BacalhauConfig {
	return BacalhauConfig{
		Environment: EnvironmentConfig{
			APIHost: "bootstrap.production.bacalhau.org",
			APIPort: 1234,
			BootstrapAddresses: []string{
				"/ip4/35.245.115.191/tcp/1235/p2p/QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL",
				"/ip4/35.245.61.251/tcp/1235/p2p/QmXaXu9N5GNetatsvwnTfQqNtSeKAD6uCmarbh3LMRYAcF",
				"/ip4/35.245.251.239/tcp/1235/p2p/QmYgxZiySj3MRkwLSL4X2MF5F9f2PMhAE3LV49XkfNL1o3",
			},
			IPFSSwarmAddresses: []string{
				"/ip4/35.245.115.191/tcp/1235/p2p/QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL",
				"/ip4/35.245.61.251/tcp/1235/p2p/QmXaXu9N5GNetatsvwnTfQqNtSeKAD6uCmarbh3LMRYAcF",
				"/ip4/35.245.251.239/tcp/1235/p2p/QmYgxZiySj3MRkwLSL4X2MF5F9f2PMhAE3LV49XkfNL1o3",
			},
		},
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
					Locality:            model.Anywhere,
					RejectStatelessJobs: false,
					AcceptNetworkedJobs: false,
					ProbeHTTP:           "",
					ProbeExec:           "",
				},
			},
		},
	}
}
