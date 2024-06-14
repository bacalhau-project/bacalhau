package node

type NetworkConfig struct {
	// NATS config for requesters to be reachable by compute nodes
	Port              int
	AdvertisedAddress string
	Orchestrators     []string

	// Storage directory for NATS features that require it
	StoreDir string

	// AuthSecret is a secret string that clients must use to connect. NATS servers
	// must supply this config, while clients can also supply it as the user part
	// of their Orchestrator URL.
	AuthSecret string

	// NATS config for requester nodes to connect with each other
	ClusterName              string
	ClusterPort              int
	ClusterAdvertisedAddress string

	// When using NATS, never set this value unless you are connecting multiple requester
	// nodes together. This should never reference this current running instance (e.g.
	// don't use localhost).
	ClusterPeers []string
}

func (c *NetworkConfig) Validate() error {
	return nil
}
