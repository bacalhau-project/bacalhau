package devstack

import (
	"context"
	"fmt"
	"math"
	"os"
	"path"
	"runtime"
	"runtime/pprof"
	"strings"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/logger"
	filecoinlotus "github.com/filecoin-project/bacalhau/pkg/publisher/filecoin_lotus"
	"github.com/filecoin-project/bacalhau/pkg/util/closer"

	"github.com/filecoin-project/bacalhau/pkg/localdb"
	"github.com/filecoin-project/bacalhau/pkg/localdb/inmemory"

	"github.com/filecoin-project/bacalhau/pkg/capacitymanager"
	"github.com/filecoin-project/bacalhau/pkg/computenode"
	"github.com/filecoin-project/bacalhau/pkg/config"
	"github.com/filecoin-project/bacalhau/pkg/ipfs"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/node"
	"github.com/filecoin-project/bacalhau/pkg/requesternode"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/transport/libp2p"
	"github.com/multiformats/go-multiaddr"
	"github.com/phayes/freeport"
	"github.com/rs/zerolog/log"
)

type DevStackOptions struct {
	NumberOfNodes        int    // Number of nodes to start in the cluster
	NumberOfBadActors    int    // Number of nodes to be bad actors
	Peer                 string // Connect node 0 to another network node
	PublicIPFSMode       bool   // Use public IPFS nodes
	LocalNetworkLotus    bool
	FilecoinUnsealedPath string
	EstuaryAPIKey        string
}
type DevStack struct {
	Nodes []*node.Node
	Lotus *LotusNode
}

func NewDevStackForRunLocal(
	ctx context.Context,
	cm *system.CleanupManager,
	count int,
	jobGPU string, //nolint:unparam // Incorrectly assumed as unused
) (*DevStack, error) {
	options := DevStackOptions{
		NumberOfNodes:  count,
		PublicIPFSMode: true,
	}

	computeNodeConfig := computenode.ComputeNodeConfig{
		JobSelectionPolicy: computenode.JobSelectionPolicy{
			Locality:            computenode.Anywhere,
			RejectStatelessJobs: false,
		}, CapacityManagerConfig: capacitymanager.Config{
			ResourceLimitTotal: model.ResourceUsageConfig{
				GPU: jobGPU,
			},
		},
	}

	return NewStandardDevStack(
		ctx,
		cm,
		options,
		computeNodeConfig,
	)
}

func NewStandardDevStack(
	ctx context.Context,
	cm *system.CleanupManager,
	options DevStackOptions,
	computeNodeConfig computenode.ComputeNodeConfig,
) (*DevStack, error) {
	return NewDevStack(ctx, cm, options, computeNodeConfig, node.NewStandardNodeDependencyInjector())
}

func NewNoopDevStack(
	ctx context.Context,
	cm *system.CleanupManager,
	options DevStackOptions,
	computeNodeConfig computenode.ComputeNodeConfig,
) (*DevStack, error) {
	return NewDevStack(ctx, cm, options, computeNodeConfig, NewNoopNodeDependencyInjector())
}

//nolint:funlen,gocyclo
func NewDevStack(
	ctx context.Context,
	cm *system.CleanupManager,
	options DevStackOptions,
	computeNodeConfig computenode.ComputeNodeConfig,
	injector node.NodeDependencyInjector,
) (*DevStack, error) {
	ctx, span := system.GetTracer().Start(ctx, "pkg/devstack.newdevstack")
	defer span.End()

	nodes := []*node.Node{}
	var lotus *LotusNode
	var err error

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

	for i := 0; i < options.NumberOfNodes; i++ {
		log.Debug().Msgf(`Creating Node #%d`, i)

		// -------------------------------------
		// IPFS
		// -------------------------------------
		var ipfsNode *ipfs.Node
		var ipfsClient *ipfs.Client

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

		ipfsClient, err = ipfsNode.Client()
		if err != nil {
			return nil, fmt.Errorf("failed to create ipfs client: %w", err)
		}

		//////////////////////////////////////
		// libp2p
		//////////////////////////////////////
		var libp2pPort int
		libp2pPort, err = freeport.GetFreePort()
		if err != nil {
			return nil, err
		}

		libp2pPeer := []multiaddr.Multiaddr{}

		if i == 0 {
			if options.Peer != "" {
				// connect 0'th node to external peer if specified
				log.Debug().Msgf("Connecting 0'th node to remote peer: %s", options.Peer)
				peerAddr, addrErr := multiaddr.NewMultiaddr(options.Peer)
				if addrErr != nil {
					return nil, fmt.Errorf("failed to parse peer address: %w", addrErr)
				}
				libp2pPeer = []multiaddr.Multiaddr{peerAddr}
			}
		} else {
			libp2pPeer, err = nodes[0].Transport.HostAddrs()
			if err != nil {
				return nil, fmt.Errorf("failed to get libp2p addresses: %w", err)
			}
			log.Debug().Msgf("Connecting to first libp2p scheduler node: %s", libp2pPeer)
		}

		transport, transportErr := libp2p.NewTransport(ctx, cm, libp2pPort, libp2pPeer)
		if transportErr != nil {
			return nil, transportErr
		}

		// add NodeID to logging context
		ctx = logger.ContextWithNodeIDLogger(ctx, transport.HostID())

		//////////////////////////////////////
		// port for API
		//////////////////////////////////////
		var apiPort int
		if os.Getenv("PREDICTABLE_API_PORT") != "" {
			apiPort = 20000 + i
		} else {
			apiPort, err = freeport.GetFreePort()
			if err != nil {
				return nil, err
			}
		}

		//////////////////////////////////////
		// metrics
		//////////////////////////////////////
		var metricsPort int
		metricsPort, err = freeport.GetFreePort()
		if err != nil {
			return nil, err
		}

		//////////////////////////////////////
		// in-memory datastore
		//////////////////////////////////////
		var datastore localdb.LocalDB
		datastore, err = inmemory.NewInMemoryDatastore()
		if err != nil {
			return nil, err
		}

		//////////////////////////////////////
		// Create and Run Node
		//////////////////////////////////////
		isBadActor := (options.NumberOfBadActors > 0) && (i >= options.NumberOfNodes-options.NumberOfBadActors)

		nodeConfig := node.NodeConfig{
			IPFSClient:           ipfsClient,
			CleanupManager:       cm,
			LocalDB:              datastore,
			Transport:            transport,
			FilecoinUnsealedPath: options.FilecoinUnsealedPath,
			EstuaryAPIKey:        options.EstuaryAPIKey,
			HostAddress:          "0.0.0.0",
			HostID:               transport.HostID(),
			APIPort:              apiPort,
			MetricsPort:          metricsPort,
			ComputeNodeConfig:    computeNodeConfig,
			RequesterNodeConfig:  requesternode.RequesterNodeConfig{},
			IsBadActor:           isBadActor,
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

		var n *node.Node
		n, err = node.NewNode(ctx, nodeConfig, injector)
		if err != nil {
			return nil, err
		}

		// Start transport layer
		err = transport.Start(ctx)
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
	// do a GC before we start profiling
	runtime.GC()

	log.Trace().Msg("============= STARTING PROFILING ============")
	// devstack always records a cpu profile, it will be generally useful.
	cpuprofile := path.Join(os.TempDir(), "bacalhau-devstack-cpu.prof")
	f, err := os.Create(cpuprofile)
	if err != nil {
		log.Fatal().Msgf("could not create CPU profile: %s", err) //nolint:gocritic
	}
	defer closer.CloseWithLogOnError("cpuprofile", f)
	if err := pprof.StartCPUProfile(f); err != nil {
		log.Fatal().Msgf("could not start CPU profile: %s", err) //nolint:gocritic
	}

	return &DevStack{
		Nodes: nodes,
		Lotus: lotus,
	}, nil
}

func createIPFSNode(ctx context.Context,
	cm *system.CleanupManager,
	publicIPFSMode bool,
	ipfsSwarmAddrs []string) (*ipfs.Node, error) {
	ctx, span := system.GetTracer().Start(ctx, "pkg/devstack.createipfsnode")
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

func (stack *DevStack) PrintNodeInfo() (string, error) {
	ctx := context.Background()

	if !config.DevstackGetShouldPrintInfo() {
		return "", nil
	}

	logString := ""
	devStackAPIPort := ""
	devStackAPIHost := "0.0.0.0"
	devStackIPFSSwarmAddress := ""

	logString += `
-----------------------------------------
-----------------------------------------
`
	for nodeIndex, node := range stack.Nodes {
		swarmAddrrs := ""
		swarmAddresses, err := node.IPFSClient.SwarmAddresses(context.Background())
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
			stack.Nodes[nodeIndex].APIServer.Host,
			nodeIndex,
			stack.Nodes[nodeIndex].APIServer.Port,
		)

		// Just setting this to the last one, really doesn't matter
		swarmAddressesList, _ := node.IPFSClient.SwarmAddresses(ctx)
		devStackIPFSSwarmAddress = strings.Join(swarmAddressesList, ",")
		devStackAPIHost = stack.Nodes[nodeIndex].APIServer.Host
		devStackAPIPort = fmt.Sprintf("%d", stack.Nodes[nodeIndex].APIServer.Port)
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

	log.Debug().Msg(logString)

	returnString := fmt.Sprintf(`
Devstack is ready!
To use the devstack, run the following commands in your shell: %s`, summaryShellVariablesString)
	return returnString, nil
}

func (stack *DevStack) GetNode(ctx context.Context, nodeID string) (
	*node.Node, error) {
	for _, node := range stack.Nodes {
		if node.Transport.HostID() == nodeID {
			return node, nil
		}
	}

	return nil, fmt.Errorf("node not found: %s", nodeID)
}

func (stack *DevStack) GetNodeIds() ([]string, error) {
	var ids []string
	for _, node := range stack.Nodes {
		ids = append(ids, node.Transport.HostID())
	}

	return ids, nil
}
