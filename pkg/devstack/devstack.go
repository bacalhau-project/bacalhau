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
	"github.com/filecoin-project/bacalhau/pkg/publicapi"
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
	"go.opentelemetry.io/otel/trace"
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
	map[storage.StorageSourceType]storage.StorageProvider,
	error,
)

type GetExecutorsFunc func(
	ipfsMultiAddress string,
	nodeIndex int,
	ctrl *controller.Controller,
) (
	map[executor.EngineType]executor.Executor,
	error,
)

type GetVerifiersFunc func(
	ipfsMultiAddress string,
	nodeIndex int,
	ctrl *controller.Controller,
) (
	map[verifier.VerifierType]verifier.Verifier,
	error,
)

func NewDevStackForRunLocal(
	cm *system.CleanupManager,
	count int,
	jobGPU string, //nolint:unparam // Incorrectly assumed as unused
) (*DevStack, error) {
	getStorageProviders := func(ipfsMultiAddress string, nodeIndex int) (map[storage.StorageSourceType]storage.StorageProvider, error) {
		return executor_util.NewStandardStorageProviders(cm, ipfsMultiAddress)
	}
	getExecutors := func(
		ipfsMultiAddress string,
		nodeIndex int,
		_ *controller.Controller,
	) (
		map[executor.EngineType]executor.Executor,
		error,
	) {
		ipfsParts := strings.Split(ipfsMultiAddress, "/")
		ipfsSuffix := ipfsParts[len(ipfsParts)-1]
		return executor_util.NewStandardExecutors(
			cm,
			ipfsMultiAddress,
			fmt.Sprintf("devstacknode%d-%s", nodeIndex, ipfsSuffix),
		)
	}
	getVerifiers := func(
		ipfsMultiAddress string,
		_ int,
		ctrl *controller.Controller,
	) (
		map[verifier.VerifierType]verifier.Verifier,
		error,
	) {
		jobLoader := func(ctx context.Context, id string) (executor.Job, error) {
			return ctrl.GetJob(ctx, id)
		}
		stateLoader := func(ctx context.Context, id string) (executor.JobState, error) {
			return ctrl.GetJobState(ctx, id)
		}
		return verifier_util.NewIPFSVerifiers(cm, ipfsMultiAddress, jobLoader, stateLoader)
	}

	return NewDevStack(
		cm,
		count, 0,
		getStorageProviders,
		getExecutors,
		getVerifiers,
		computenode.ComputeNodeConfig{
			JobSelectionPolicy: computenode.JobSelectionPolicy{
				Locality:            computenode.Anywhere,
				RejectStatelessJobs: false,
			}, CapacityManagerConfig: capacitymanager.Config{
				ResourceLimitTotal: capacitymanager.ResourceUsageConfig{
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
	cm *system.CleanupManager,
	count, _ int, //nolint:unparam // Incorrectly assumed as unused
	getStorageProviders GetStorageProvidersFunc,
	getExecutors GetExecutorsFunc,
	getVerifiers GetVerifiersFunc,
	//nolint:gocritic
	config computenode.ComputeNodeConfig,
	peer string,
	publicIPFSMode bool,
) (*DevStack, error) {
	ctx, span := newSpan("NewDevStack")
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
			ipfsNode, err = ipfs.NewNode(cm, []string{})
			if err != nil {
				return nil, fmt.Errorf("failed to create ipfs node: %w", err)
			}
		} else {
			ipfsNode, err = ipfs.NewLocalNode(cm, ipfsSwarmAddrs)
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

		transport, err := libp2p.NewTransport(cm, libp2pPort, []string{libp2pPeer})
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
			cm,
			inmemoryDatastore,
			transport,
			storageProviders,
		)
		if err != nil {
			return nil, err
		}

		executors, err := getExecutors(ipfsAPIAddrs[0], i, ctrl)
		if err != nil {
			return nil, err
		}

		verifiers, err := getVerifiers(ipfsAPIAddrs[0], i, ctrl)
		if err != nil {
			return nil, err
		}

		//////////////////////////////////////
		// Requestor node
		//////////////////////////////////////
		requesterNode, err := requesternode.NewRequesterNode(
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
			cm,
			ctrl,
			executors,
			verifiers,
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
			"0.0.0.0",
			apiPort,
			ctrl,
			verifiers,
		)
		go func(ctx context.Context) {
			var gerr error // don't capture outer scope
			if gerr = apiServer.ListenAndServe(ctx, cm); gerr != nil {
				panic(gerr) // if api server can't run, devstack should stop
			}
		}(context.Background())

		log.Debug().Msgf("public API server started: 0.0.0.0:%d", apiPort)

		//////////////////////////////////////
		// metrics
		//////////////////////////////////////

		metricsPort, err := freeport.GetFreePort()
		if err != nil {
			return nil, err
		}

		// TODO: #393 Why is ctx unused if it's passed in? Shouldn't cm do something with it?
		go func(ctx context.Context) { //nolint:unparam // Ok to be unused?
			var gerr error // don't capture outer scope
			if gerr = system.ListenAndServeMetrics(cm, metricsPort); gerr != nil {
				log.Error().Msgf("Cannot serve metrics: %v", err)
			}
		}(context.Background())

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
		logString += fmt.Sprintf(`
-------------------------------
node %d
-------------------------------

export BACALHAU_IPFS_API_PORT_%d=%d
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

func (stack *DevStack) AddFileToNodes(nodeCount int, filePath string) (string, error) {
	var res string
	for i, node := range stack.Nodes {
		if i >= nodeCount {
			continue
		}

		cid, err := node.IpfsClient.Put(context.Background(), filePath)
		if err != nil {
			return "", fmt.Errorf("error adding file to node %d: %v", i, err)
		}

		log.Debug().Msgf("Added cid '%s' to ipfs node '%s'", cid, node.IpfsNode.ID())
		res = strings.TrimSpace(cid)
	}

	return res, nil
}

func (stack *DevStack) AddTextToNodes(nodeCount int, fileContent []byte) (string, error) {
	testDir, err := ioutil.TempDir("", "bacalhau-test")
	if err != nil {
		return "", err
	}

	testFilePath := fmt.Sprintf("%s/test.txt", testDir)
	err = os.WriteFile(testFilePath, fileContent, util.OS_USER_RW)
	if err != nil {
		return "", err
	}

	return stack.AddFileToNodes(nodeCount, testFilePath)
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

func newSpan(name string) (context.Context, trace.Span) {
	return system.Span(context.Background(), "devstack", name)
}
