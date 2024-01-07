package devstack

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/bacalhau-project/bacalhau/pkg/util/multiaddresses"
	"github.com/imdario/mergo"
	"github.com/libp2p/go-libp2p/core/host"
	"github.com/multiformats/go-multiaddr"
	"github.com/phayes/freeport"
	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/ipfs"
	bac_libp2p "github.com/bacalhau-project/bacalhau/pkg/libp2p"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/node"
	"github.com/bacalhau-project/bacalhau/pkg/repo"
	"github.com/bacalhau-project/bacalhau/pkg/routing"
	"github.com/bacalhau-project/bacalhau/pkg/system"
)

const (
	DefaultLibp2pKeySize = 2048
)

type DevStackOptions struct {
	NumberOfHybridNodes        int    // Number of nodes to start in the cluster
	NumberOfRequesterOnlyNodes int    // Number of nodes to start in the cluster
	NumberOfComputeOnlyNodes   int    // Number of nodes to start in the cluster
	NumberOfBadComputeActors   int    // Number of compute nodes to be bad actors
	NumberOfBadRequesterActors int    // Number of requester nodes to be bad actors
	Peer                       string // Connect node 0 to another network node
	PublicIPFSMode             bool   // Use public IPFS nodes
	CPUProfilingFile           string
	MemoryProfilingFile        string
	DisabledFeatures           node.FeatureConfig
	AllowListedLocalPaths      []string // Local paths that are allowed to be mounted into jobs
	NodeInfoPublisherInterval  routing.NodeInfoPublisherIntervalConfig
	ExecutorPlugins            bool   // when true pluggable executors will be used.
	ConfigurationRepo          string // A custom config repo
	UseNATS                    bool   // Use NATS transport
}

func (o *DevStackOptions) Options() []ConfigOption {
	opts := []ConfigOption{
		WithNumberOfHybridNodes(o.NumberOfHybridNodes),
		WithNumberOfRequesterOnlyNodes(o.NumberOfRequesterOnlyNodes),
		WithNumberOfComputeOnlyNodes(o.NumberOfComputeOnlyNodes),
		WithNumberOfBadComputeActors(o.NumberOfBadComputeActors),
		WithNumberOfBadRequesterActors(o.NumberOfBadRequesterActors),
		WithPeer(o.Peer),
		WithPublicIPFSMode(o.PublicIPFSMode),
		WithCPUProfilingFile(o.CPUProfilingFile),
		WithMemoryProfilingFile(o.MemoryProfilingFile),
		WithDisabledFeatures(o.DisabledFeatures),
		WithAllowListedLocalPaths(o.AllowListedLocalPaths),
		WithNodeInfoPublisherInterval(o.NodeInfoPublisherInterval),
		WithExecutorPlugins(o.ExecutorPlugins),
	}
	if o.UseNATS {
		opts = append(opts, WithNATS(o.UseNATS))
	}
	return opts
}

type DevStack struct {
	Nodes          []*node.Node
	PublicIPFSMode bool
}

//nolint:funlen,gocyclo
func Setup(
	ctx context.Context,
	cm *system.CleanupManager,
	fsRepo *repo.FsRepo,
	opts ...ConfigOption,
) (*DevStack, error) {
	stackConfig, err := defaultDevStackConfig()
	if err != nil {
		return nil, fmt.Errorf("creating devstack defaults: %w", err)
	}
	for _, opt := range opts {
		opt(stackConfig)
	}

	if err := stackConfig.Validate(); err != nil {
		return nil, fmt.Errorf("validating devstask config: %w", err)
	}

	log.Ctx(ctx).Info().Object("Config", stackConfig).Msg("Starting Devstack")
	ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/devstack.Setup")
	defer span.End()

	var nodes []*node.Node
	orchestratorAddrs := make([]string, 0)
	clusterPeersAddrs := make([]string, 0)

	totalNodeCount := stackConfig.NumberOfHybridNodes + stackConfig.NumberOfRequesterOnlyNodes + stackConfig.NumberOfComputeOnlyNodes
	requesterNodeCount := stackConfig.NumberOfHybridNodes + stackConfig.NumberOfRequesterOnlyNodes
	computeNodeCount := stackConfig.NumberOfHybridNodes + stackConfig.NumberOfComputeOnlyNodes

	if requesterNodeCount == 0 {
		return nil, fmt.Errorf("at least one requester node is required")
	}

	for i := 0; i < totalNodeCount; i++ {
		nodeID := fmt.Sprintf("node-%d", i)
		ctx = logger.ContextWithNodeIDLogger(ctx, nodeID)

		isRequesterNode := i < requesterNodeCount
		isComputeNode := (totalNodeCount - i) <= computeNodeCount
		log.Ctx(ctx).Debug().Msgf(`Creating Node #%d as {RequesterNode: %t, ComputeNode: %t}`, i+1, isRequesterNode, isComputeNode)

		// ////////////////////////////////////
		// IPFS
		// ////////////////////////////////////

		var ipfsSwarmAddresses []string
		if i > 0 {
			addresses, err := nodes[0].IPFSClient.SwarmAddresses(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to get ipfs swarm addresses: %w", err)
			}
			// Only use a single address as libp2p seems to have concurrency issues, like two nodes not able to finish
			// connecting/joining topics, when using multiple addresses for a single host.
			// All the IPFS nodes are running within the same process, so connecting over localhost will be fine.
			ipfsSwarmAddresses = append(ipfsSwarmAddresses, addresses[0])
		}

		ipfsNode, err := createIPFSNode(ctx, cm, stackConfig.PublicIPFSMode, ipfsSwarmAddresses)
		if err != nil {
			return nil, fmt.Errorf("failed to create ipfs node: %w", err)
		}

		// ////////////////////////////////////
		// Transport layer (NATS or Libp2p)
		// ////////////////////////////////////
		var swarmPort int
		if os.Getenv("PREDICTABLE_API_PORT") != "" {
			const startSwarmPort = 4222 // 4222 is the default NATS port
			swarmPort = startSwarmPort + i
		} else {
			swarmPort, err = freeport.GetFreePort()
			if err != nil {
				return nil, err
			}
		}
		clusterConfig := node.NetworkConfig{
			UseNATS:       stackConfig.UseNATS,
			Orchestrators: orchestratorAddrs,
			Port:          swarmPort,
			ClusterPeers:  clusterPeersAddrs,
		}

		if stackConfig.UseNATS {
			var clusterPort int
			if os.Getenv("PREDICTABLE_API_PORT") != "" {
				const startClusterPort = 6222
				clusterPort = startClusterPort + i
			} else {
				clusterPort, err = freeport.GetFreePort()
				if err != nil {
					return nil, err
				}
			}

			if isRequesterNode {
				clusterConfig.ClusterName = "devstack"
				clusterConfig.ClusterPort = clusterPort
				orchestratorAddrs = append(orchestratorAddrs, fmt.Sprintf("0.0.0.0:%d", swarmPort))
				clusterPeersAddrs = append(clusterPeersAddrs, fmt.Sprintf("0.0.0.0:%d", clusterPort))
			}
		} else {
			if i == 0 {
				if stackConfig.Peer != "" {
					clusterConfig.ClusterPeers = append(clusterConfig.ClusterPeers, stackConfig.Peer)
				}
			} else {
				p2pAddr, err := multiaddr.NewMultiaddr("/p2p/" + nodes[0].Libp2pHost.ID().String())
				if err != nil {
					return nil, err
				}
				addresses := multiaddresses.SortLocalhostFirst(nodes[0].Libp2pHost.Addrs())
				clusterConfig.ClusterPeers = append(clusterConfig.ClusterPeers, addresses[0].Encapsulate(p2pAddr).String())
			}

			clusterConfig.Libp2pHost, err = createLibp2pHost(ctx, cm, swarmPort)
			if err != nil {
				return nil, err
			}

			// nodeID must match the libp2p host ID
			nodeID = clusterConfig.Libp2pHost.ID().String()
			ctx = logger.ContextWithNodeIDLogger(ctx, nodeID)
		}

		// ////////////////////////////////////
		// port for API
		// ////////////////////////////////////
		apiPort := uint16(0)
		if os.Getenv("PREDICTABLE_API_PORT") != "" {
			const startPort = 20000
			apiPort = uint16(startPort + i)
		}

		// ////////////////////////////////////
		// Create and Run Node
		// ////////////////////////////////////

		// here is where we can parse string based CLI stackConfig
		// into more meaningful model.FailureInjectionConfig values
		isBadComputeActor := (stackConfig.NumberOfBadComputeActors > 0) && (i >= computeNodeCount-stackConfig.NumberOfBadComputeActors)
		isBadRequesterActor := (stackConfig.NumberOfBadRequesterActors > 0) && (i >= requesterNodeCount-stackConfig.NumberOfBadRequesterActors)

		if isBadComputeActor {
			stackConfig.ComputeConfig.FailureInjectionConfig.IsBadActor = isBadComputeActor
		}

		if isBadRequesterActor {
			stackConfig.RequesterConfig.FailureInjectionConfig.IsBadActor = isBadRequesterActor
		}

		nodeInfoPublisherInterval := stackConfig.NodeInfoPublisherInterval
		if nodeInfoPublisherInterval.IsZero() {
			nodeInfoPublisherInterval = node.TestNodeInfoPublishConfig
		}

		nodeConfig := node.NodeConfig{
			NodeID:              nodeID,
			IPFSClient:          ipfsNode.Client(),
			CleanupManager:      cm,
			HostAddress:         "0.0.0.0",
			APIPort:             apiPort,
			ComputeConfig:       stackConfig.ComputeConfig,
			RequesterNodeConfig: stackConfig.RequesterConfig,
			IsComputeNode:       isComputeNode,
			IsRequesterNode:     isRequesterNode,
			Labels: map[string]string{
				"id":   nodeID,
				"name": fmt.Sprintf("node-%d", i),
				"env":  "devstack",
			},
			DependencyInjector:        stackConfig.NodeDependencyInjector,
			DisabledFeatures:          stackConfig.DisabledFeatures,
			AllowListedLocalPaths:     stackConfig.AllowListedLocalPaths,
			NodeInfoPublisherInterval: nodeInfoPublisherInterval,
			FsRepo:                    fsRepo,
			NodeInfoStoreTTL:          stackConfig.NodeInfoStoreTTL,
			NetworkConfig:             clusterConfig,
		}

		if isRequesterNode {
			// If we are setting up the requester node, then we should set the TLS
			// settings needed for AutoCert.
			nodeConfig.RequesterAutoCert = config.ServerAutoCertDomain()
			nodeConfig.RequesterAutoCertCache = config.GetAutoCertCachePath()

			cert, key := config.GetRequesterCertificateSettings()
			nodeConfig.RequesterTLSCertificateFile = cert
			nodeConfig.RequesterTLSKeyFile = key
		}

		// allow overriding configs of some nodes
		if i < len(stackConfig.NodeOverrides) {
			originalConfig := nodeConfig
			nodeConfig = stackConfig.NodeOverrides[i]
			err = mergo.Merge(&nodeConfig, originalConfig)
			if err != nil {
				return nil, err
			}
		}

		var n *node.Node
		n, err = node.NewNode(ctx, nodeConfig)
		if err != nil {
			return nil, err
		}

		// start the node
		err = n.Start(ctx)
		if err != nil {
			return nil, err
		}

		nodes = append(nodes, n)
	}

	// only start profiling after we've set everything up!
	profiler := startProfiling(ctx, stackConfig.CPUProfilingFile, stackConfig.MemoryProfilingFile)
	if profiler != nil {
		cm.RegisterCallbackWithContext(profiler.Close)
	}

	return &DevStack{
		Nodes:          nodes,
		PublicIPFSMode: stackConfig.PublicIPFSMode,
	}, nil
}

func createLibp2pHost(ctx context.Context, cm *system.CleanupManager, port int) (host.Host, error) {
	var err error

	// TODO(forrest): [devstack] Refactor the devstack s.t. each node has its own repo and config.
	// previously the config would generate a key using the host port as the postfix
	// this is not longer the case as a node should have a single libp2p key, but since
	// all devstack nodes share a repo we will get a self dial error if we use the same
	// key from the config for each devstack node. The solution here is to refactor the
	// the devstack such that all nodes in the stack have their own repos and configuration
	// rather than rely on global values and one off key gen via the config.

	privKey, err := bac_libp2p.GeneratePrivateKey(DefaultLibp2pKeySize)
	if err != nil {
		return nil, err
	}

	libp2pHost, err := bac_libp2p.NewHost(port, privKey)
	if err != nil {
		return nil, fmt.Errorf("error creating libp2p host: %w", err)
	}

	return libp2pHost, nil
}

func createIPFSNode(ctx context.Context,
	cm *system.CleanupManager,
	publicIPFSMode bool,
	ipfsSwarmAddresses []string) (*ipfs.Node, error) {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/devstack.createIPFSNode")
	defer span.End()
	// ////////////////////////////////////
	// IPFS
	// ////////////////////////////////////
	return ipfs.NewNodeWithConfig(ctx, cm, types.IpfsConfig{SwarmAddresses: ipfsSwarmAddresses, PrivateInternal: !publicIPFSMode})
}

//nolint:funlen
func (stack *DevStack) PrintNodeInfo(ctx context.Context, fsRepo *repo.FsRepo, cm *system.CleanupManager) (string, error) {
	if !config.DevstackGetShouldPrintInfo() {
		return "", nil
	}

	logString := ""
	devStackAPIPort := fmt.Sprintf("%d", stack.Nodes[0].APIServer.Port)
	devStackAPIHost := stack.Nodes[0].APIServer.Address
	devStackIPFSSwarmAddress := ""
	var devstackPeerAddrs []string

	// TODO remove this it's wrong and never printed, nothing sets the env vars its printing
	logString += `
-----------------------------------------
-----------------------------------------
`

	requesterOnlyNodes := 0
	computeOnlyNodes := 0
	hybridNodes := 0
	for nodeIndex, node := range stack.Nodes {
		swarmAddrrs := ""
		swarmAddresses, err := node.IPFSClient.SwarmAddresses(ctx)
		if err != nil {
			return "", fmt.Errorf("cannot get swarm addresses for node %d", nodeIndex)
		} else {
			swarmAddrrs = strings.Join(swarmAddresses, ",")
		}

		peerConnect := fmt.Sprintf("/ip4/%s/tcp/%d/http", node.APIServer.Address, node.APIServer.Port)
		devstackPeerAddrs = append(devstackPeerAddrs, peerConnect)

		logString += fmt.Sprintf(`
export BACALHAU_IPFS_%d=%s
export BACALHAU_IPFS_SWARM_ADDRESSES_%d=%s
export BACALHAU_PEER_CONNECT_%d=%s
export BACALHAU_API_HOST_%d=%s
export BACALHAU_API_PORT_%d=%d`,
			nodeIndex,
			node.IPFSClient.APIAddress(),
			nodeIndex,
			swarmAddrrs,
			nodeIndex,
			peerConnect,
			nodeIndex,
			stack.Nodes[nodeIndex].APIServer.Address,
			nodeIndex,
			stack.Nodes[nodeIndex].APIServer.Port,
		)

		requesterOnlyNodes += boolToInt(node.IsRequesterNode() && !node.IsComputeNode())
		computeOnlyNodes += boolToInt(node.IsComputeNode() && !node.IsRequesterNode())
		hybridNodes += boolToInt(node.IsRequesterNode() && node.IsComputeNode())

		// Just setting this to the last one, really doesn't matter
		swarmAddressesList, _ := node.IPFSClient.SwarmAddresses(ctx)
		devStackIPFSSwarmAddress = strings.Join(swarmAddressesList, ",")
	}

	summaryBuilder := strings.Builder{}
	summaryBuilder.WriteString(fmt.Sprintf(
		"export %s=%s\n",
		config.KeyAsEnvVar(types.NodeIPFSSwarmAddresses),
		devStackIPFSSwarmAddress,
	))
	summaryBuilder.WriteString(fmt.Sprintf(
		"export %s=%s\n",
		config.KeyAsEnvVar(types.NodeClientAPIHost),
		devStackAPIHost,
	))
	summaryBuilder.WriteString(fmt.Sprintf(
		"export %s=%s\n",
		config.KeyAsEnvVar(types.NodeClientAPIPort),
		devStackAPIPort,
	))
	summaryBuilder.WriteString(fmt.Sprintf(
		"export %s=%s\n",
		config.KeyAsEnvVar(types.NodeLibp2pPeerConnect),
		strings.Join(devstackPeerAddrs, ","),
	))

	// Just convenience below - print out the last of the nodes information as the global variable
	summaryShellVariablesString := summaryBuilder.String()

	ripath, err := fsRepo.WriteRunInfo(ctx, summaryShellVariablesString)
	if err != nil {
		return "", err
	}
	cm.RegisterCallback(func() error {
		return os.Remove(ripath)
	})

	if !stack.PublicIPFSMode {
		summaryBuilder.WriteString(
			"\nBy default devstack is not running on the public IPFS network.\n" +
				"If you wish to connect devstack to the public IPFS network add the --public-ipfs flag.\n" +
				"You can also run a new IPFS daemon locally and connect it to Bacalhau using:\n\n",
		)
		summaryBuilder.WriteString(
			fmt.Sprintf("ipfs swarm connect $%s", config.KeyAsEnvVar(types.NodeIPFSSwarmAddresses)),
		)
	}

	log.Ctx(ctx).Debug().Msg(logString)

	returnString := fmt.Sprintf(`
Devstack is ready!
No. of requester only nodes: %d
No. of compute only nodes: %d
No. of hybrid nodes: %d
To use the devstack, run the following commands in your shell:

%s

The above variables were also written to this file (will be deleted when devstack exits): %s`,
		requesterOnlyNodes,
		computeOnlyNodes,
		hybridNodes,
		summaryBuilder.String(),
		ripath)
	return returnString, nil
}

func (stack *DevStack) GetNode(_ context.Context, nodeID string) (
	*node.Node, error) {
	for _, node := range stack.Nodes {
		if node.ID == nodeID {
			return node, nil
		}
	}

	return nil, fmt.Errorf("node not found: %s", nodeID)
}
func (stack *DevStack) IPFSClients() []ipfs.Client {
	clients := make([]ipfs.Client, 0, len(stack.Nodes))
	for _, node := range stack.Nodes {
		clients = append(clients, node.IPFSClient)
	}
	return clients
}

func (stack *DevStack) GetNodeIds() []string {
	var ids []string
	for _, node := range stack.Nodes {
		ids = append(ids, node.ID)
	}
	return ids
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
