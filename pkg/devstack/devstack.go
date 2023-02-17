package devstack

import (
	"context"
	"fmt"
	"math"
	"os"
	"strings"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/jobstore/inmemory"
	"github.com/filecoin-project/bacalhau/pkg/libp2p"
	"github.com/filecoin-project/bacalhau/pkg/logger"
	filecoinlotus "github.com/filecoin-project/bacalhau/pkg/publisher/filecoin_lotus"
	"github.com/imdario/mergo"
	"github.com/libp2p/go-libp2p/core/host"

	"github.com/filecoin-project/bacalhau/pkg/config"
	"github.com/filecoin-project/bacalhau/pkg/ipfs"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/node"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/multiformats/go-multiaddr"
	"github.com/phayes/freeport"
	"github.com/rs/zerolog/log"
)

type DevStackOptions struct {
	NumberOfHybridNodes        int    // Number of nodes to start in the cluster
	NumberOfRequesterOnlyNodes int    // Number of nodes to start in the cluster
	NumberOfComputeOnlyNodes   int    // Number of nodes to start in the cluster
	NumberOfBadComputeActors   int    // Number of compute nodes to be bad actors
	NumberOfBadRequesterActors int    // Number of requester nodes to be bad actors
	Peer                       string // Connect node 0 to another network node
	PublicIPFSMode             bool   // Use public IPFS nodes
	LocalNetworkLotus          bool
	FilecoinUnsealedPath       string
	EstuaryAPIKey              string
	SimulatorAddr              string // if this is set, we will use the simulator transport
	SimulatorMode              bool   // if this is set, the first node will be a simulator node and will use the simulator transport
	CPUProfilingFile           string
	MemoryProfilingFile        string
}
type DevStack struct {
	Nodes          []*node.Node
	Lotus          *LotusNode
	PublicIPFSMode bool
}

func NewDevStackForRunLocal(
	ctx context.Context,
	cm *system.CleanupManager,
	count int,
	jobGPU uint64, //nolint:unparam // Incorrectly assumed as unused
) (*DevStack, error) {
	options := DevStackOptions{
		NumberOfHybridNodes: count,
		PublicIPFSMode:      true,
	}

	computeConfig := node.NewComputeConfigWith(node.ComputeConfigParams{
		TotalResourceLimits: model.ResourceUsageData{GPU: jobGPU},
		JobSelectionPolicy: model.JobSelectionPolicy{
			Locality:            model.Anywhere,
			RejectStatelessJobs: false,
		},
	})

	return NewStandardDevStack(
		ctx,
		cm,
		options,
		computeConfig,
		node.NewRequesterConfigWithDefaults(),
	)
}

func NewStandardDevStack(
	ctx context.Context,
	cm *system.CleanupManager,
	options DevStackOptions,
	computeConfig node.ComputeConfig,
	requesterNodeConfig node.RequesterConfig,
) (*DevStack, error) {
	return NewDevStack(ctx, cm, options, computeConfig, requesterNodeConfig, node.NewStandardNodeDependencyInjector())
}

func NewNoopDevStack(
	ctx context.Context,
	cm *system.CleanupManager,
	options DevStackOptions,
	computeConfig node.ComputeConfig,
	requesterNodeConfig node.RequesterConfig,
) (*DevStack, error) {
	return NewDevStack(ctx, cm, options, computeConfig, requesterNodeConfig, NewNoopNodeDependencyInjector())
}

//nolint:funlen,gocyclo
func NewDevStack(
	ctx context.Context,
	cm *system.CleanupManager,
	options DevStackOptions,
	computeConfig node.ComputeConfig,
	requesterNodeConfig node.RequesterConfig,
	injector node.NodeDependencyInjector,
	nodeOverrides ...node.NodeConfig,
) (*DevStack, error) {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/devstack.NewDevStack")
	defer span.End()

	var nodes []*node.Node
	var lotus *LotusNode
	var err error
	var simulatorAddr multiaddr.Multiaddr
	var simulatorNodeID string

	if options.SimulatorAddr != "" {
		simulatorAddr, err = multiaddr.NewMultiaddr(options.SimulatorAddr)
		if err != nil {
			return nil, fmt.Errorf("failed to parse simulator address: %w", err)
		}
		simulatorNodeID, err = simulatorAddr.ValueForProtocol(multiaddr.P_P2P)
		if err != nil {
			return nil, fmt.Errorf("failed to extract p2p protocoll from simulator address: %w", err)
		}
	}

	if options.LocalNetworkLotus {
		lotus, err = newLotusNode(ctx) //nolint:govet
		if err != nil {
			return nil, err
		}

		cm.RegisterCallback(lotus.Close)

		if err := lotus.start(ctx); err != nil { //nolint:govet
			return nil, err
		}
	}

	totalNodeCount := options.NumberOfHybridNodes + options.NumberOfRequesterOnlyNodes + options.NumberOfComputeOnlyNodes
	requesterNodeCount := options.NumberOfHybridNodes + options.NumberOfRequesterOnlyNodes
	computeNodeCount := options.NumberOfHybridNodes + options.NumberOfComputeOnlyNodes

	if requesterNodeCount == 0 {
		return nil, fmt.Errorf("at least one requester node is required")
	}
	for i := 0; i < totalNodeCount; i++ {
		isRequesterNode := i < requesterNodeCount
		isComputeNode := (totalNodeCount - i) <= computeNodeCount
		log.Debug().Msgf(`Creating Node #%d as {RequesterNode: %t, ComputeNode: %t}`, i+1, isRequesterNode, isComputeNode)

		// -------------------------------------
		// IPFS
		// -------------------------------------
		var ipfsNode *ipfs.Node
		var ipfsClient ipfs.Client

		var ipfsSwarmAddrs []string
		if i > 0 {
			ipfsSwarmAddrs, err = nodes[0].IPFSClient.SwarmAddresses(ctx)
			if err != nil {
				return nil, fmt.Errorf("failed to get ipfs swarm addresses: %w", err)
			}
		}

		ipfsNode, err = createIPFSNode(ctx, cm, options.PublicIPFSMode, ipfsSwarmAddrs)
		if err != nil {
			return nil, fmt.Errorf("failed to create ipfs node: %w", err)
		}

		ipfsClient = ipfsNode.Client()

		var libp2pHost host.Host
		var libp2pPort int
		var libp2pPeer []multiaddr.Multiaddr
		if simulatorAddr != nil {
			libp2pPeer = append(libp2pPeer, simulatorAddr)
		}

		//////////////////////////////////////
		// libp2p
		//////////////////////////////////////
		libp2pPort, err := freeport.GetFreePort()
		if err != nil {
			return nil, err
		}

		if i == 0 {
			if options.Peer != "" {
				// connect 0'th node to external peer if specified
				log.Debug().Msgf("Connecting 0'th node to remote peer: %s", options.Peer)
				peerAddr, addrErr := multiaddr.NewMultiaddr(options.Peer)
				if addrErr != nil {
					return nil, fmt.Errorf("failed to parse peer address: %w", addrErr)
				}
				libp2pPeer = append(libp2pPeer, peerAddr)
			}
		} else {
			for _, addrs := range nodes[0].Host.Addrs() {
				p2pAddr, p2pAddrErr := multiaddr.NewMultiaddr("/p2p/" + nodes[0].Host.ID().String())
				if p2pAddrErr != nil {
					return nil, p2pAddrErr
				}
				libp2pPeer = append(libp2pPeer, addrs.Encapsulate(p2pAddr))
			}
			if err != nil {
				return nil, fmt.Errorf("failed to get libp2p addresses: %w", err)
			}
			log.Debug().Msgf("Connecting to first libp2p requester node: %s", libp2pPeer)
		}

		libp2pHost, err = libp2p.NewHost(libp2pPort)
		if err != nil {
			return nil, err
		}
		cm.RegisterCallback(func() error {
			return libp2pHost.Close()
		})

		// add NodeID to logging context
		ctx = logger.ContextWithNodeIDLogger(ctx, libp2pHost.ID().String())

		//////////////////////////////////////
		// port for API
		//////////////////////////////////////
		apiPort := 0
		if os.Getenv("PREDICTABLE_API_PORT") != "" {
			apiPort = 20000 + i
		}

		//////////////////////////////////////
		// in-memory datastore
		//////////////////////////////////////
		datastore := inmemory.NewJobStore()

		//////////////////////////////////////
		// Create and Run Node
		//////////////////////////////////////

		// here is where we can parse string based CLI options
		// into more meaningful model.SimulatorConfig values
		isBadComputeActor := (options.NumberOfBadComputeActors > 0) && (i >= computeNodeCount-options.NumberOfBadComputeActors)
		isBadRequesterActor := (options.NumberOfBadRequesterActors > 0) && (i >= requesterNodeCount-options.NumberOfBadRequesterActors)

		if isBadComputeActor {
			computeConfig.SimulatorConfig.IsBadActor = isBadComputeActor
		}

		if isBadRequesterActor {
			requesterNodeConfig.SimulatorConfig.IsBadActor = isBadRequesterActor
		}

		// If we are running in a simulator mode, and didn't pass in a node ID, then the first node will be the simulator node
		if options.SimulatorMode && simulatorAddr == nil {
			p2pAddr, addrError := multiaddr.NewMultiaddr("/p2p/" + libp2pHost.ID().String())
			if err != nil {
				return nil, addrError
			}
			simulatorAddr = libp2pHost.Addrs()[0].Encapsulate(p2pAddr)
			simulatorNodeID = libp2pHost.ID().String()
		}

		nodeConfig := node.NodeConfig{
			IPFSClient:           ipfsClient,
			CleanupManager:       cm,
			JobStore:             datastore,
			Host:                 libp2pHost,
			FilecoinUnsealedPath: options.FilecoinUnsealedPath,
			EstuaryAPIKey:        options.EstuaryAPIKey,
			HostAddress:          "0.0.0.0",
			APIPort:              apiPort,
			ComputeConfig:        computeConfig,
			RequesterNodeConfig:  requesterNodeConfig,
			SimulatorNodeID:      simulatorNodeID,
			IsComputeNode:        isComputeNode,
			IsRequesterNode:      isRequesterNode,
			Labels: map[string]string{
				"name": fmt.Sprintf("node-%d", i),
				"id":   libp2pHost.ID().String(),
				"env":  "devstack",
			},
		}

		if lotus != nil {
			nodeConfig.LotusConfig = &filecoinlotus.PublisherConfig{
				StorageDuration: 24 * 24 * time.Hour,
				PathDir:         lotus.PathDir,
				UploadDir:       lotus.UploadDir,
				// devstack will only be talking to a single node, so don't bother filtering based on ping
				// as the ping may be quite large while it is trying to run everything
				MaximumPing: time.Duration(math.MaxInt64),
			}
		}

		// allow overriding configs of some nodes
		if i < len(nodeOverrides) {
			originalConfig := nodeConfig
			nodeConfig = nodeOverrides[i]
			err = mergo.Merge(&nodeConfig, originalConfig)
			if err != nil {
				return nil, err
			}
		}

		var n *node.Node
		n, err = node.NewNode(ctx, nodeConfig, injector)
		if err != nil {
			return nil, err
		}

		// Start transport layer
		err = libp2p.ConnectToPeers(ctx, libp2pHost, libp2pPeer)
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
	profiler := StartProfiling(options.CPUProfilingFile, options.MemoryProfilingFile)
	if profiler != nil {
		cm.RegisterCallback(profiler.Close)
	}

	return &DevStack{
		Nodes:          nodes,
		Lotus:          lotus,
		PublicIPFSMode: options.PublicIPFSMode,
	}, nil
}

func createIPFSNode(ctx context.Context,
	cm *system.CleanupManager,
	publicIPFSMode bool,
	ipfsSwarmAddrs []string) (*ipfs.Node, error) {
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
		ipfsNode, err = ipfs.NewLocalNode(ctx, cm, ipfsSwarmAddrs)
		if err != nil {
			return nil, fmt.Errorf("failed to create ipfs node: %w", err)
		}
	}
	return ipfsNode, nil
}

func (stack *DevStack) PrintNodeInfo(ctx context.Context) (string, error) {
	if !config.DevstackGetShouldPrintInfo() {
		return "", nil
	}

	logString := ""
	devStackAPIPort := fmt.Sprintf("%d", stack.Nodes[0].APIServer.Port)
	devStackAPIHost := stack.Nodes[0].APIServer.Address
	devStackIPFSSwarmAddress := ""

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

		logString += fmt.Sprintf(`
export BACALHAU_IPFS_%d=%s
export BACALHAU_IPFS_SWARM_ADDRESSES_%d=%s
export BACALHAU_API_HOST_%d=%s
export BACALHAU_API_PORT_%d=%d`,
			nodeIndex,
			node.IPFSClient.APIAddress(),
			nodeIndex,
			swarmAddrrs,
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
export BACALHAU_API_PORT=%s`,
		devStackIPFSSwarmAddress,
		devStackAPIHost,
		devStackAPIPort,
	)

	if stack.Lotus != nil {
		summaryShellVariablesString += fmt.Sprintf(`
export LOTUS_PATH=%s
export LOTUS_UPLOAD_DIR=%s`, stack.Lotus.PathDir, stack.Lotus.UploadDir)
	}

	if config.DevstackShouldWriteEnvFile() {
		err := os.WriteFile(config.DevstackEnvFile(), []byte(summaryShellVariablesString), 0600) //nolint:gomnd
		if err != nil {
			log.Err(err).Msgf("Failed to write file %s", config.DevstackEnvFile())
			return "", err
		}
	}

	if !stack.PublicIPFSMode {
		summaryShellVariablesString += `

By default devstack is not running on the public IPFS network.
If you wish to connect devstack to the public IPFS network add the --public-ipfs flag.
You can also run a new IPFS daemon locally and connect it to Bacalhau using:

ipfs swarm connect $BACALHAU_IPFS_SWARM_ADDRESSES`
	}

	log.Debug().Msg(logString)

	returnString := fmt.Sprintf(`
Devstack is ready!
No. of requester only nodes: %d
No. of compute only nodes: %d
No. of hybrid nodes: %d
To use the devstack, run the following commands in your shell: %s`,
		requesterOnlyNodes,
		computeOnlyNodes,
		hybridNodes,
		summaryShellVariablesString)
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

func (stack *DevStack) GetNodeIds() ([]string, error) {
	var ids []string
	for _, node := range stack.Nodes {
		ids = append(ids, node.Host.ID().String())
	}
	return ids, nil
}

func boolToInt(b bool) int {
	if b {
		return 1
	}
	return 0
}
