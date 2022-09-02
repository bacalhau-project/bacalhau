package devstack

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"runtime"
	"runtime/pprof"
	"strings"

	"github.com/filecoin-project/bacalhau/pkg/capacitymanager"
	"github.com/filecoin-project/bacalhau/pkg/computenode"
	"github.com/filecoin-project/bacalhau/pkg/config"
	"github.com/filecoin-project/bacalhau/pkg/controller"
	"github.com/filecoin-project/bacalhau/pkg/executor"
	executor_util "github.com/filecoin-project/bacalhau/pkg/executor/util"
	"github.com/filecoin-project/bacalhau/pkg/ipfs"
	"github.com/filecoin-project/bacalhau/pkg/localdb/inmemory"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/publicapi"
	"github.com/filecoin-project/bacalhau/pkg/publisher"
	publisher_util "github.com/filecoin-project/bacalhau/pkg/publisher/util"
	"github.com/filecoin-project/bacalhau/pkg/requesternode"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/storage/util"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/transport/libp2p"
	"github.com/filecoin-project/bacalhau/pkg/verifier"
	verifier_util "github.com/filecoin-project/bacalhau/pkg/verifier/util"
	"github.com/ipfs/go-datastore"
	"github.com/phayes/freeport"
	"github.com/rs/zerolog/log"
)

type DevStackNode struct {
	ComputeNode   *computenode.ComputeNode
	RequesterNode *requesternode.RequesterNode
	Transport     *libp2p.LibP2PTransport
	Controller    *controller.Controller
	Datastore     datastore.Datastore

	IpfsNode   *ipfs.Node
	IpfsClient *ipfs.Client
	Libp2pPort int
	APIServer  *publicapi.APIServer
}

type DevStack struct {
	Nodes []*DevStackNode
}

type GetStorageProvidersFunc func(
	ipfsMultiAddress string,
	nodeIndex int,
) (
	map[model.StorageSourceType]storage.StorageProvider,
	error,
)

type GetExecutorsFunc func(
	ipfsMultiAddress string,
	nodeIndex int,
	isBadActor bool,
	ctrl *controller.Controller,
) (
	map[model.EngineType]executor.Executor,
	error,
)

type GetVerifiersFunc func(
	transport *libp2p.LibP2PTransport,
	nodeIndex int,
	ctrl *controller.Controller,
) (
	map[model.VerifierType]verifier.Verifier,
	error,
)

type GetPublishersFunc func(
	ipfsMultiAddress string,
	nodeIndex int,
	ctrl *controller.Controller,
) (
	map[model.PublisherType]publisher.Publisher,
	error,
)

func NewDevStackForRunLocal(
	ctx context.Context, cm *system.CleanupManager,

	count int,
	jobGPU string, //nolint:unparam // Incorrectly assumed as unused
) (*DevStack, error) {
	getStorageProviders := func(ipfsMultiAddress string, nodeIndex int) (map[model.StorageSourceType]storage.StorageProvider, error) {
		return executor_util.NewStandardStorageProviders(ctx, cm, executor_util.StandardStorageProviderOptions{
			IPFSMultiaddress: ipfsMultiAddress,
		})
	}
	getExecutors := func(
		ipfsMultiAddress string,
		nodeIndex int,
		isBadActor bool,
		_ *controller.Controller,
	) (
		map[model.EngineType]executor.Executor,
		error,
	) {
		ipfsParts := strings.Split(ipfsMultiAddress, "/")
		ipfsSuffix := ipfsParts[len(ipfsParts)-1]
		return executor_util.NewStandardExecutors(
			ctx,
			cm,
			executor_util.StandardExecutorOptions{
				DockerID: fmt.Sprintf("devstacknode%d-%s", nodeIndex, ipfsSuffix),
				Storage: executor_util.StandardStorageProviderOptions{
					IPFSMultiaddress: ipfsMultiAddress,
				},
			},
		)
	}
	getVerifiers := func(
		transport *libp2p.LibP2PTransport,
		_ int,
		ctrl *controller.Controller,
	) (
		map[model.VerifierType]verifier.Verifier,
		error,
	) {
		return verifier_util.NewStandardVerifiers(
			ctx,
			cm,
			ctrl.GetStateResolver(),
			transport.Encrypt,
			transport.Decrypt,
		)
	}
	getPublishers := func(
		ipfsMultiAddress string,
		nodeIndex int,
		ctrl *controller.Controller,
	) (
		map[model.PublisherType]publisher.Publisher,
		error,
	) {
		return publisher_util.NewIPFSPublishers(ctx, cm, ctrl.GetStateResolver(), ipfsMultiAddress)
	}

	return NewDevStack(
		ctx,
		cm,
		count, 0,
		getStorageProviders,
		getExecutors,
		getVerifiers,
		getPublishers,
		computenode.ComputeNodeConfig{
			JobSelectionPolicy: computenode.JobSelectionPolicy{
				Locality:            computenode.Anywhere,
				RejectStatelessJobs: false,
			}, CapacityManagerConfig: capacitymanager.Config{
				ResourceLimitTotal: model.ResourceUsageConfig{
					GPU: jobGPU,
				},
			},
		},
		"",
		true,
	)
}

//nolint:funlen,gocyclo
func NewDevStack(
	ctx context.Context,
	cm *system.CleanupManager,
	count, badActors int, //nolint:unparam // Incorrectly assumed as unused
	getStorageProviders GetStorageProvidersFunc,
	getExecutors GetExecutorsFunc,
	getVerifiers GetVerifiersFunc,
	getPublishers GetPublishersFunc,
	//nolint:gocritic
	config computenode.ComputeNodeConfig,
	peer string,
	publicIPFSMode bool,
) (*DevStack, error) {
	t := system.GetTracer()
	_, span := t.Start(ctx, "newdevstack")
	defer span.End()

	nodes := []*DevStackNode{}
	for i := 0; i < count; i++ {
		log.Debug().Msgf(`Creating Node #%d`, i)

		//////////////////////////////////////
		// IPFS
		//////////////////////////////////////
		var err error
		var ipfsSwarmAddrs []string
		var ipfsNode *ipfs.Node

		if i > 0 {
			ipfsSwarmAddrs, err = nodes[0].IpfsNode.SwarmAddresses()
			if err != nil {
				return nil, fmt.Errorf("failed to get ipfs swarm addresses: %w", err)
			}
		}

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

		ipfsClient, err := ipfsNode.Client()
		if err != nil {
			return nil, fmt.Errorf("failed to create ipfs client: %w", err)
		}

		ipfsAPIAddrs, err := ipfsNode.APIAddresses()
		if err != nil {
			return nil, fmt.Errorf("failed to get ipfs api addresses: %w", err)
		}
		if len(ipfsAPIAddrs) == 0 { // should never happen
			return nil, fmt.Errorf("devstack ipfs node has no api addresses")
		}

		inmemoryDatastore, err := inmemory.NewInMemoryDatastore()
		if err != nil {
			return nil, err
		}

		libp2pPort, err := freeport.GetFreePort()
		if err != nil {
			return nil, err
		}

		libp2pPeer := ""

		if i == 0 {
			if peer != "" {
				// connect 0'th node to external peer if specified
				log.Debug().Msgf("Connecting 0'th node to remote peer: %s", peer)
				libp2pPeer = peer
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
			libp2pPeer = fmt.Sprintf("/ip4/127.0.0.1/tcp/%d/p2p/%s", firstNode.Libp2pPort, libp2pHostID)
			log.Debug().Msgf("Connecting to first libp2p scheduler node: %s", libp2pPeer)
		}

		transport, err := libp2p.NewTransport(ctx, cm, libp2pPort, []string{libp2pPeer})
		if err != nil {
			return nil, err
		}

		//////////////////////////////////////
		// Storage, executors and verifiers
		//////////////////////////////////////
		storageProviders, err := getStorageProviders(ipfsAPIAddrs[0], i)
		if err != nil {
			return nil, err
		}

		//////////////////////////////////////
		// Controller
		//////////////////////////////////////
		ctrl, err := controller.NewController(
			ctx,
			cm,
			inmemoryDatastore,
			transport,
			storageProviders,
		)
		if err != nil {
			return nil, err
		}

		isBadActor := false

		if badActors > 0 {
			isBadActor = i >= count-badActors
		}

		executors, err := getExecutors(ipfsAPIAddrs[0], i, isBadActor, ctrl)
		if err != nil {
			return nil, err
		}

		verifiers, err := getVerifiers(transport, i, ctrl)
		if err != nil {
			return nil, err
		}

		publishers, err := getPublishers(ipfsAPIAddrs[0], i, ctrl)
		if err != nil {
			return nil, err
		}

		//////////////////////////////////////
		// Requestor node
		//////////////////////////////////////
		requesterNode, err := requesternode.NewRequesterNode(
			ctx,
			cm,
			ctrl,
			verifiers,
			requesternode.RequesterNodeConfig{},
		)
		if err != nil {
			return nil, err
		}

		//////////////////////////////////////
		// Compute node
		//////////////////////////////////////
		computeNode, err := computenode.NewComputeNode(
			ctx,
			cm,
			ctrl,
			executors,
			verifiers,
			publishers,
			config,
		)
		if err != nil {
			return nil, err
		}

		//////////////////////////////////////
		// JSON RPC
		//////////////////////////////////////

		// predictable port for API
		var apiPort int
		if os.Getenv("PREDICTABLE_API_PORT") != "" {
			apiPort = 20000 + i
		} else {
			apiPort, err = freeport.GetFreePort()
			if err != nil {
				return nil, err
			}
		}

		apiServer := publicapi.NewServer(
			ctx,
			"0.0.0.0",
			apiPort,
			ctrl,
			publishers,
		)
		go func(ctx context.Context) {
			var gerr error // don't capture outer scope
			if gerr = apiServer.ListenAndServe(ctx, cm); gerr != nil {
				panic(gerr) // if api server can't run, devstack should stop
			}
		}(ctx)

		log.Debug().Msgf("public API server started: 0.0.0.0:%d", apiPort)

		//////////////////////////////////////
		// metrics
		//////////////////////////////////////

		metricsPort, err := freeport.GetFreePort()
		if err != nil {
			return nil, err
		}

		go func(ctx context.Context) { //nolint:unparam
			var gerr error // don't capture outer scope
			if gerr = system.ListenAndServeMetrics(ctx, cm, metricsPort); gerr != nil {
				log.Error().Msgf("Cannot serve metrics: %v", err)
			}
		}(ctx)

		//////////////////////////////////////
		// intra-connections
		//////////////////////////////////////

		go func(ctx context.Context) {
			if err = ctrl.Start(ctx); err != nil {
				panic(err) // if controller can't run, devstack should stop
			}
			if err = transport.Start(ctx); err != nil {
				panic(err) // if transport can't run, devstack should stop
			}
		}(context.Background())

		log.Debug().Msgf("libp2p server started: %d", libp2pPort)

		devStackNode := &DevStackNode{
			ComputeNode:   computeNode,
			RequesterNode: requesterNode,
			Transport:     transport,
			IpfsNode:      ipfsNode,
			IpfsClient:    ipfsClient,
			APIServer:     apiServer,
			Libp2pPort:    libp2pPort,
		}

		nodes = append(nodes, devStackNode)
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

func (stack *DevStack) PrintNodeInfo() {
	if !config.DevstackGetShouldPrintInfo() {
		return
	}

	logString := ""
	devStackAPIPort := ""
	devStackAPIHost := "0.0.0.0"

	for nodeIndex, node := range stack.Nodes {
		swarmAddrrs := ""
		swarmAddresses, err := node.IpfsNode.SwarmAddresses()
		if err != nil {
			log.Error().Msgf("Cannot get swarm addresses for node %d", nodeIndex)
		} else {
			swarmAddrrs = strings.Join(swarmAddresses, ",")
		}

		logString += fmt.Sprintf(`
-------------------------------
node %d
-------------------------------

export BACALHAU_IPFS_API_PORT_%d=%d
export BACALHAU_IPFS_SWARM_ADDRESSES_%d=%s
export BACALHAU_IPFS_PATH_%d=%s
export BACALHAU_API_HOST_%d=%s
export BACALHAU_API_PORT_%d=%d
cid=$(ipfs --api /ip4/127.0.0.1/tcp/%d add --quiet ./testdata/grep_file.txt)
curl -XPOST http://127.0.0.1:%d/api/v0/id
`,
			nodeIndex,
			nodeIndex,
			node.IpfsNode.APIPort,
			nodeIndex,
			swarmAddrrs,
			nodeIndex,
			node.IpfsNode.RepoPath,
			nodeIndex,
			stack.Nodes[nodeIndex].APIServer.Host,
			nodeIndex,
			stack.Nodes[nodeIndex].APIServer.Port,
			node.IpfsNode.APIPort,
			node.IpfsNode.APIPort,
		)
	}

	logString += `
-----------------------------------------
-----------------------------------------
`
	for nodeIndex, node := range stack.Nodes {
		logString += fmt.Sprintf(`
export BACALHAU_IPFS_PATH_%d=%s
export BACALHAU_API_HOST_%d=%s
export BACALHAU_API_PORT_%d=%d`,
			nodeIndex,
			node.IpfsNode.RepoPath,
			nodeIndex,
			stack.Nodes[nodeIndex].APIServer.Host,
			nodeIndex,
			stack.Nodes[nodeIndex].APIServer.Port,
		)

		// Just setting this to the last one, really doesn't matter
		devStackAPIHost = stack.Nodes[nodeIndex].APIServer.Host
		devStackAPIPort = fmt.Sprintf("%d", stack.Nodes[nodeIndex].APIServer.Port)
	}

	// Just convenience below - print out the last of the nodes information as the global variable
	logString += fmt.Sprintf(`
export BACALHAU_API_HOST=%s
export BACALHAU_API_PORT=%s`,
		devStackAPIHost,
		devStackAPIPort,
	)
	log.Debug().Msg(logString)
}

func (stack *DevStack) AddFileToNodes(ctx context.Context, nodeCount int, filePath string) (string, error) {
	var res string
	for i, node := range stack.Nodes {
		if i >= nodeCount {
			continue
		}

		cid, err := node.IpfsClient.Put(ctx, filePath)
		if err != nil {
			return "", fmt.Errorf("error adding file to node %d: %v", i, err)
		}

		log.Debug().Msgf("Added cid '%s' to ipfs node '%s'", cid, node.IpfsNode.ID())
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
	*DevStackNode, error) {
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
