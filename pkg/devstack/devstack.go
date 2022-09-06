package devstack

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"runtime"
	"runtime/pprof"
	"strconv"
	"strings"

	"github.com/filecoin-project/bacalhau/pkg/capacitymanager"
	"github.com/filecoin-project/bacalhau/pkg/computenode"
	"github.com/filecoin-project/bacalhau/pkg/config"
	"github.com/filecoin-project/bacalhau/pkg/ipfs"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/node"
	"github.com/filecoin-project/bacalhau/pkg/requesternode"
	"github.com/filecoin-project/bacalhau/pkg/storage/util"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/transport"
	"github.com/filecoin-project/bacalhau/pkg/transport/libp2p"
	"github.com/phayes/freeport"
	"github.com/rs/zerolog/log"
)

type DevStackOptions struct {
	NumberOfNodes        int    // Number of nodes to start in the cluster
	NumberOfBadActors    int    // Number of nodes to be bad actors
	Peer                 string // Connect node 0 to another network node
	PublicIPFSMode       bool   // Use public IPFS nodes
	FilecoinUnsealedPath string
	EstuaryAPIKey        string
}
type DevStack struct {
	Nodes []*node.Node
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
	return NewDevStack(ctx, cm, options, computeNodeConfig, node.NewStandardNodeDepdencyInjector())
}

func NewNoopDevStack(
	ctx context.Context,
	cm *system.CleanupManager,
	options DevStackOptions,
	computeNodeConfig computenode.ComputeNodeConfig,
) (*DevStack, error) {
	return NewDevStack(ctx, cm, options, computeNodeConfig, NewNoopNodeDepdencyInjector())
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
	var firstNodeLibp2pPort int
	var err error

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

		libp2pPeer := ""

		if i == 0 {
			firstNodeLibp2pPort = libp2pPort
			if options.Peer != "" {
				// connect 0'th node to external peer if specified
				log.Debug().Msgf("Connecting 0'th node to remote peer: %s", options.Peer)
				libp2pPeer = options.Peer
			}
		} else {
			var libp2pHostID string
			// connect the libp2p scheduler node
			firstNode := nodes[0]

			// get the libp2p id of the first scheduler node
			libp2pHostID, err = firstNode.Transport.HostID(ctx)
			if err != nil {
				return nil, err
			}

			// connect this scheduler to the first
			libp2pPeer = fmt.Sprintf("/ip4/127.0.0.1/tcp/%d/p2p/%s", firstNodeLibp2pPort, libp2pHostID)
			log.Debug().Msgf("Connecting to first libp2p scheduler node: %s", libp2pPeer)
		}

		var transport transport.Transport
		transport, err = libp2p.NewTransport(ctx, cm, libp2pPort, []string{libp2pPeer})
		if err != nil {
			return nil, err
		}

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
		// Create and Run Node
		//////////////////////////////////////
		isBadActor := (options.NumberOfBadActors > 0) && (i >= options.NumberOfNodes-options.NumberOfBadActors)

		nodeConfig := node.NodeConfig{
			IPFSClient:           ipfsClient,
			CleanupManager:       cm,
			Transport:            transport,
			FilecoinUnsealedPath: options.FilecoinUnsealedPath,
			EstuaryAPIKey:        options.EstuaryAPIKey,
			HostAddress:          "0.0.0.0",
			HostID:               strconv.Itoa(i),
			APIPort:              apiPort,
			MetricsPort:          metricsPort,
			ComputeNodeConfig:    computeNodeConfig,
			RequesterNodeConfig:  requesternode.RequesterNodeConfig{},
			IsBadActor:           isBadActor,
		}

		var n *node.Node
		n, err = node.NewNode(ctx, nodeConfig, injector)
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
	cpuprofile := "/tmp/bacalhau-devstack-cpu.prof"
	f, err := os.Create(cpuprofile)
	if err != nil {
		log.Fatal().Msgf("could not create CPU profile: %s", err) //nolint:gocritic
	}
	defer f.Close()
	if err := pprof.StartCPUProfile(f); err != nil {
		log.Fatal().Msgf("could not start CPU profile: %s", err) //nolint:gocritic
	}

	return &DevStack{
		Nodes: nodes,
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

func (stack *DevStack) PrintNodeInfo() {
	ctx := context.Background()

	if !config.DevstackGetShouldPrintInfo() {
		return
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
			log.Error().Msgf("Cannot get swarm addresses for node %d", nodeIndex)
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

	log.Debug().Msg(logString)

	log.Info().Msg("Devstack is ready!")
	log.Info().Msg("To use the devstack, run the following commands in your shell:")
	log.Info().Msg(summaryShellVariablesString)
}

func (stack *DevStack) AddFileToNodes(ctx context.Context, nodeCount int, filePath string) (string, error) {
	var res string
	for i, node := range stack.Nodes {
		if i >= nodeCount {
			continue
		}

		cid, err := node.IPFSClient.Put(ctx, filePath)
		if err != nil {
			return "", fmt.Errorf("error adding file to node %d: %v", i, err)
		}

		log.Debug().Msgf("Added cid '%s' to ipfs node '%s'", cid, node.IPFSClient.APIAddress())
		res = strings.TrimSpace(cid)
	}

	return res, nil
}

func (stack *DevStack) AddTextToNodes(ctx context.Context, nodeCount int, fileContent []byte) (string, error) {
	testDir, err := ioutil.TempDir("", "bacalhau-test")
	if err != nil {
		return "", err
	}

	testFilePath := fmt.Sprintf("%s/test.txt", testDir)
	err = os.WriteFile(testFilePath, fileContent, util.OS_USER_RW)
	if err != nil {
		return "", err
	}

	return stack.AddFileToNodes(ctx, nodeCount, testFilePath)
}

func (stack *DevStack) GetNode(ctx context.Context, nodeID string) (
	*node.Node, error) {
	for _, node := range stack.Nodes {
		id, err := node.Transport.HostID(ctx)
		if err != nil {
			return nil, err
		}

		if id == nodeID {
			return node, nil
		}
	}

	return nil, fmt.Errorf("node not found: %s", nodeID)
}

func (stack *DevStack) GetNodeIds() ([]string, error) {
	ids := []string{}
	for _, node := range stack.Nodes {
		id, err := node.Transport.HostID(context.Background())
		if err != nil {
			return ids, err
		}
		ids = append(ids, id)
	}

	return ids, nil
}

func (stack *DevStack) GetShortIds() ([]string, error) {
	ids, err := stack.GetNodeIds()
	if err != nil {
		return ids, err
	}
	shortids := []string{}
	for _, id := range ids {
		shortids = append(shortids, system.ShortID(id))
	}
	return shortids, nil
}
