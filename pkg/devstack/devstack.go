package devstack

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/imdario/mergo"
	"github.com/multiformats/go-multiaddr"
	"github.com/phayes/freeport"
	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/routing"

	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/ipfs"
	"github.com/bacalhau-project/bacalhau/pkg/libp2p"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/util/multiaddresses"

	"github.com/bacalhau-project/bacalhau/pkg/node"
	"github.com/bacalhau-project/bacalhau/pkg/system"
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
	ExecutorPlugins            bool // when true pluggable executors will be used.
}

func (o *DevStackOptions) Options() []ConfigOption {
	return []ConfigOption{
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
}

type DevStack struct {
	Nodes          []*node.Node
	PublicIPFSMode bool
}

//nolint:funlen,gocyclo
func Setup(
	ctx context.Context,
	cm *system.CleanupManager,
	opts ...ConfigOption,
) (*DevStack, error) {
	stackConfig := defaultDevStackConfig()
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

	totalNodeCount := stackConfig.NumberOfHybridNodes + stackConfig.NumberOfRequesterOnlyNodes + stackConfig.NumberOfComputeOnlyNodes
	requesterNodeCount := stackConfig.NumberOfHybridNodes + stackConfig.NumberOfRequesterOnlyNodes
	computeNodeCount := stackConfig.NumberOfHybridNodes + stackConfig.NumberOfComputeOnlyNodes

	if requesterNodeCount == 0 {
		return nil, fmt.Errorf("at least one requester node is required")
	}
	for i := 0; i < totalNodeCount; i++ {
		isRequesterNode := i < requesterNodeCount
		isComputeNode := (totalNodeCount - i) <= computeNodeCount
		log.Ctx(ctx).Debug().Msgf(`Creating Node #%d as {RequesterNode: %t, ComputeNode: %t}`, i+1, isRequesterNode, isComputeNode)

		//////////////////////////////////////
		// IPFS
		//////////////////////////////////////

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

		//////////////////////////////////////
		// libp2p
		//////////////////////////////////////
		var libp2pPeer []multiaddr.Multiaddr
		libp2pPort, err := freeport.GetFreePort()
		if err != nil {
			return nil, err
		}

		if i == 0 {
			if stackConfig.Peer != "" {
				// connect 0'th node to external peer if specified
				log.Ctx(ctx).Debug().Msgf("Connecting 0'th node to remote peer: %s", stackConfig.Peer)
				peerAddr, addrErr := multiaddr.NewMultiaddr(stackConfig.Peer)
				if addrErr != nil {
					return nil, fmt.Errorf("failed to parse peer address: %w", addrErr)
				}
				libp2pPeer = append(libp2pPeer, peerAddr)
			}
		} else {
			p2pAddr, err := multiaddr.NewMultiaddr("/p2p/" + nodes[0].Host.ID().String())
			if err != nil {
				return nil, err
			}
			addresses := multiaddresses.SortLocalhostFirst(nodes[0].Host.Addrs())
			// Only use a single address as libp2p seems to have concurrency issues, like two nodes not able to finish
			// connecting/joining topics, when using multiple addresses for a single host.
			libp2pPeer = append(libp2pPeer, addresses[0].Encapsulate(p2pAddr))
			log.Ctx(ctx).Debug().Msgf("Connecting to first libp2p requester node: %s", libp2pPeer)
		}

		libp2pHost, err := libp2p.NewHost(libp2pPort)
		if err != nil {
			return nil, err
		}
		cm.RegisterCallback(libp2pHost.Close)

		// add NodeID to logging context
		ctx = logger.ContextWithNodeIDLogger(ctx, libp2pHost.ID().String())

		//////////////////////////////////////
		// port for API
		//////////////////////////////////////
		apiPort := uint16(0)
		if os.Getenv("PREDICTABLE_API_PORT") != "" {
			const startPort = 20000
			apiPort = uint16(startPort + i)
		}

		//////////////////////////////////////
		// Create and Run Node
		//////////////////////////////////////

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
			IPFSClient:          ipfsNode.Client(),
			CleanupManager:      cm,
			Host:                libp2pHost,
			HostAddress:         "0.0.0.0",
			APIPort:             apiPort,
			ComputeConfig:       stackConfig.ComputeConfig,
			RequesterNodeConfig: stackConfig.RequesterConfig,
			IsComputeNode:       isComputeNode,
			IsRequesterNode:     isRequesterNode,
			Labels: map[string]string{
				"name": fmt.Sprintf("node-%d", i),
				"id":   libp2pHost.ID().String(),
				"env":  "devstack",
			},
			DependencyInjector:        stackConfig.NodeDependencyInjector,
			DisabledFeatures:          stackConfig.DisabledFeatures,
			AllowListedLocalPaths:     stackConfig.AllowListedLocalPaths,
			NodeInfoPublisherInterval: nodeInfoPublisherInterval,
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

		// Start transport layer
		err = libp2p.ConnectToPeersContinuouslyWithRetryDuration(ctx, cm, libp2pHost, libp2pPeer, 2*time.Second)
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

func createIPFSNode(ctx context.Context,
	cm *system.CleanupManager,
	publicIPFSMode bool,
	ipfsSwarmAddresses []string) (*ipfs.Node, error) {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/devstack.createIPFSNode")
	defer span.End()
	//////////////////////////////////////
	// IPFS
	//////////////////////////////////////
	var err error
	var ipfsNode *ipfs.Node

	if publicIPFSMode {
		ipfsNode, err = ipfs.NewNode(ctx, cm, []string{})
		if err != nil {
			return nil, fmt.Errorf("failed to create ipfs node: %w", err)
		}
	} else {
		ipfsNode, err = ipfs.NewLocalNode(ctx, cm, ipfsSwarmAddresses)
		if err != nil {
			return nil, fmt.Errorf("failed to create ipfs node: %w", err)
		}
	}
	return ipfsNode, nil
}

//nolint:funlen
func (stack *DevStack) PrintNodeInfo(ctx context.Context, cm *system.CleanupManager) (string, error) {
	if !config.DevstackGetShouldPrintInfo() {
		return "", nil
	}

	logString := ""
	devStackAPIPort := fmt.Sprintf("%d", stack.Nodes[0].APIServer.Port)
	devStackAPIHost := stack.Nodes[0].APIServer.Address
	devStackIPFSSwarmAddress := ""
	var devstackPeerAddrs []string

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

		var libp2pPeer []string
		for _, addrs := range node.Host.Addrs() {
			p2pAddr, p2pAddrErr := multiaddr.NewMultiaddr("/p2p/" + node.Host.ID().String())
			if p2pAddrErr != nil {
				return "", p2pAddrErr
			}
			libp2pPeer = append(libp2pPeer, addrs.Encapsulate(p2pAddr).String())
		}
		devstackPeerAddr := strings.Join(libp2pPeer, ",")
		if len(libp2pPeer) > 0 {
			chosen := false
			preferredAddress := config.PreferredAddress()
			if preferredAddress != "" {
				for _, addr := range libp2pPeer {
					if strings.Contains(addr, preferredAddress) {
						devstackPeerAddrs = append(devstackPeerAddrs, addr)
						chosen = true
						break
					}
				}
			}

			if !chosen {
				// only add one of the addrs for this peer and we will choose the first
				// in the absence of a preference
				devstackPeerAddrs = append(devstackPeerAddrs, libp2pPeer[0])
			}
		}

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
			devstackPeerAddr,
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

	// Just convenience below - print out the last of the nodes information as the global variable
	summaryShellVariablesString := fmt.Sprintf(`
export BACALHAU_IPFS_SWARM_ADDRESSES=%s
export BACALHAU_API_HOST=%s
export BACALHAU_API_PORT=%s
export BACALHAU_PEER_CONNECT=%s`,
		devStackIPFSSwarmAddress,
		devStackAPIHost,
		devStackAPIPort,
		strings.Join(devstackPeerAddrs, ","),
	)

	err := config.WriteRunInfoFile(ctx, summaryShellVariablesString)
	if err != nil {
		return "", err
	}
	cm.RegisterCallback(config.CleanupRunInfoFile)

	if !stack.PublicIPFSMode {
		summaryShellVariablesString += `

By default devstack is not running on the public IPFS network.
If you wish to connect devstack to the public IPFS network add the --public-ipfs flag.
You can also run a new IPFS daemon locally and connect it to Bacalhau using:

ipfs swarm connect $BACALHAU_IPFS_SWARM_ADDRESSES`
	}

	log.Ctx(ctx).Debug().Msg(logString)

	returnString := fmt.Sprintf(`
Devstack is ready!
No. of requester only nodes: %d
No. of compute only nodes: %d
No. of hybrid nodes: %d
To use the devstack, run the following commands in your shell: %s

The above variables were also written to this file (will be deleted when devstack exits): %s`,
		requesterOnlyNodes,
		computeOnlyNodes,
		hybridNodes,
		summaryShellVariablesString,
		config.GetRunInfoFilePath())
	return returnString, nil
}

func (stack *DevStack) GetNode(_ context.Context, nodeID string) (
	*node.Node, error) {
	for _, node := range stack.Nodes {
		if node.Host.ID().String() == nodeID {
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
		ids = append(ids, node.Host.ID().String())
	}
	return ids
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
