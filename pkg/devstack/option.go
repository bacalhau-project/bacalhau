package devstack

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/imdario/mergo"
	"github.com/multiformats/go-multiaddr"
	"github.com/phayes/freeport"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/jobstore/inmemory"
	"github.com/bacalhau-project/bacalhau/pkg/libp2p"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/node"
	"github.com/bacalhau-project/bacalhau/pkg/routing"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/bacalhau-project/bacalhau/pkg/util/multiaddresses"
)

type ConfigOption = func(cfg *DevStackConfig)

func defaultDevStackConfig() *DevStackConfig {
	return &DevStackConfig{
		ComputeConfig:          node.NewComputeConfigWithDefaults(),
		RequesterConfig:        node.NewRequesterConfigWithDefaults(),
		NodeDependencyInjector: node.NodeDependencyInjector{},
		NodeOverrides:          nil,

		NumberOfRequesterOnlyNodes: 1,
		NumberOfComputeOnlyNodes:   3,
		NumberOfBadComputeActors:   0,
		Peer:                       "",
		PublicIPFSMode:             false,
		EstuaryAPIKey:              os.Getenv("ESTUARY_API_KEY"),
		CPUProfilingFile:           "",
		MemoryProfilingFile:        "",
		NodeInfoPublisherInterval:  node.TestNodeInfoPublishConfig,

		NumberOfBadRequesterActors: 0,
		NumberOfHybridNodes:        0,
		DisabledFeatures:           node.FeatureConfig{},
		AllowListedLocalPaths:      nil,
		ExecutorPlugins:            false,
	}
}

type DevStackConfig struct {
	ComputeConfig          node.ComputeConfig
	RequesterConfig        node.RequesterConfig
	NodeDependencyInjector node.NodeDependencyInjector
	NodeOverrides          []node.NodeConfig

	// DevStackOptions
	NumberOfHybridNodes        int    // Number of nodes to start in the cluster
	NumberOfRequesterOnlyNodes int    // Number of nodes to start in the cluster
	NumberOfComputeOnlyNodes   int    // Number of nodes to start in the cluster
	NumberOfBadComputeActors   int    // Number of compute nodes to be bad actors
	NumberOfBadRequesterActors int    // Number of requester nodes to be bad actors
	Peer                       string // Connect node 0 to another network node
	PublicIPFSMode             bool   // Use public IPFS nodes
	EstuaryAPIKey              string
	CPUProfilingFile           string
	MemoryProfilingFile        string
	DisabledFeatures           node.FeatureConfig
	AllowListedLocalPaths      []string // Local paths that are allowed to be mounted into jobs
	NodeInfoPublisherInterval  routing.NodeInfoPublisherIntervalConfig
	ExecutorPlugins            bool // when true pluggable executors will be used.
}

func (o *DevStackConfig) MarshalZerologObject(e *zerolog.Event) {
	e.Int("HybridNodes", o.NumberOfHybridNodes).
		Int("RequesterOnlyNodes", o.NumberOfRequesterOnlyNodes).
		Int("ComputeOnlyNodes", o.NumberOfComputeOnlyNodes).
		Int("BadComputeActors", o.NumberOfBadComputeActors).
		Int("BadRequesterActors", o.NumberOfBadRequesterActors).
		Str("Peer", o.Peer).
		Bool("PublicIPFSMode", o.PublicIPFSMode).
		Str("EstuaryAPIKey", o.EstuaryAPIKey).
		Str("CPUProfilingFile", o.CPUProfilingFile).
		Str("MemoryProfilingFile", o.MemoryProfilingFile).
		Str("DisabledFeatures", fmt.Sprintf("%v", o.DisabledFeatures)).
		Strs("AllowListedLocalPaths", o.AllowListedLocalPaths).
		Str("NodeInfoPublisherInterval", fmt.Sprintf("%v", o.NodeInfoPublisherInterval)).
		Bool("ExecutorPlugins", o.ExecutorPlugins)
}

func WithNodeOverrides(overrides ...node.NodeConfig) ConfigOption {
	return func(cfg *DevStackConfig) {
		cfg.NodeOverrides = overrides
	}
}

func WithDependencyInjector(injector node.NodeDependencyInjector) ConfigOption {
	return func(cfg *DevStackConfig) {
		cfg.NodeDependencyInjector = injector
	}
}

func WithComputeConfig(computeCfg node.ComputeConfig) ConfigOption {
	return func(cfg *DevStackConfig) {
		cfg.ComputeConfig = computeCfg
	}
}

func WithRequesterConfig(requesterConfig node.RequesterConfig) ConfigOption {
	return func(cfg *DevStackConfig) {
		cfg.RequesterConfig = requesterConfig
	}
}

func WithNumberOfHybridNodes(count int) ConfigOption {
	return func(cfg *DevStackConfig) {
		cfg.NumberOfHybridNodes = count
	}
}

func WithPublicIPFSMode(enabled bool) ConfigOption {
	return func(cfg *DevStackConfig) {
		cfg.PublicIPFSMode = enabled
	}
}

func WithNumberOfRequesterOnlyNodes(count int) ConfigOption {
	return func(cfg *DevStackConfig) {
		cfg.NumberOfRequesterOnlyNodes = count
	}
}

func WithNumberOfComputeOnlyNodes(count int) ConfigOption {
	return func(cfg *DevStackConfig) {
		cfg.NumberOfComputeOnlyNodes = count
	}
}

func WithNumberOfBadComputeActors(count int) ConfigOption {
	return func(cfg *DevStackConfig) {
		cfg.NumberOfBadComputeActors = count
	}
}

func WithNumberOfBadRequesterActors(count int) ConfigOption {
	return func(cfg *DevStackConfig) {
		cfg.NumberOfBadRequesterActors = count
	}
}

func WithPeer(p string) ConfigOption {
	return func(cfg *DevStackConfig) {
		cfg.Peer = p
	}
}

func WithEstuaryAPIKey(key string) ConfigOption {
	return func(cfg *DevStackConfig) {
		cfg.EstuaryAPIKey = key
	}
}

func WithCPUProfilingFile(path string) ConfigOption {
	return func(cfg *DevStackConfig) {
		cfg.CPUProfilingFile = path
	}
}

func WithMemoryProfilingFile(path string) ConfigOption {
	return func(cfg *DevStackConfig) {
		cfg.MemoryProfilingFile = path
	}
}

func WithDisabledFeatures(disable node.FeatureConfig) ConfigOption {
	return func(cfg *DevStackConfig) {
		cfg.DisabledFeatures = disable
	}
}

func WithAllowListedLocalPaths(paths []string) ConfigOption {
	return func(cfg *DevStackConfig) {
		cfg.AllowListedLocalPaths = paths
	}
}

func WithNodeInfoPublisherInterval(interval routing.NodeInfoPublisherIntervalConfig) ConfigOption {
	return func(cfg *DevStackConfig) {
		cfg.NodeInfoPublisherInterval = interval
	}
}

func WithExecutorPlugins(enabled bool) ConfigOption {
	return func(cfg *DevStackConfig) {
		cfg.ExecutorPlugins = enabled
	}
}

func NewDevStack(
	ctx context.Context,
	cm *system.CleanupManager,
	opts ...ConfigOption,
) (*DevStack, error) {
	stackConfig := defaultDevStackConfig()
	for _, opt := range opts {
		opt(stackConfig)
	}
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	log.Ctx(ctx).Info().Object("Config", stackConfig).Msg("Starting Devstack")
	ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/devstack.NewDevStack")
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
			JobStore:            inmemory.NewJobStore(),
			Host:                libp2pHost,
			EstuaryAPIKey:       stackConfig.EstuaryAPIKey,
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
