package devstack

import (
	"context"
	"fmt"

	"github.com/filecoin-project/bacalhau/pkg/compute_node"
	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/executor/docker"
	"github.com/filecoin-project/bacalhau/pkg/executor/docker/ipfs"
	ipfs_cli "github.com/filecoin-project/bacalhau/pkg/ipfs/cli"
	ipfs_devstack "github.com/filecoin-project/bacalhau/pkg/ipfs/devstack"
	"github.com/filecoin-project/bacalhau/pkg/jsonrpc"
	"github.com/filecoin-project/bacalhau/pkg/requestor_node"
	"github.com/filecoin-project/bacalhau/pkg/scheduler/libp2p"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/phayes/freeport"
	"github.com/rs/zerolog/log"
)

type DevStackNode struct {
	Ctx           context.Context
	ComputeNode   *compute_node.ComputeNode
	RequesterNode *requestor_node.RequesterNode
	IpfsNode      *ipfs_devstack.IPFSDevServer
	IpfsCli       *ipfs_cli.IPFSCli
	SchedulerNode *libp2p.Libp2pScheduler
	JSONRpcNode   *jsonrpc.JSONRpcServer
}

type DevStack struct {
	Nodes []*DevStackNode
}

func NewDockerIPFSExecutors(ctx context.Context, ipfsMultiAddress string) (map[string]executor.Executor, error) {
	executors := map[string]executor.Executor{}
	ipfsStorage, err := ipfs.NewDockerStorageIPFS(ctx, ipfsMultiAddress)
	if err != nil {
		return executors, err
	}
	dockerExecutor, err := docker.NewDockerExecutor(ctx, map[string]storage.StorageProvider{
		"ipfs": ipfsStorage,
	})
	if err != nil {
		return executors, err
	}
	executors["docker"] = dockerExecutor
	return executors, nil
}

func NewDevStack(
	ctx context.Context,
	count, badActors int,
	getExecutors func(ipfsMultiAddress string) (map[string]executor.Executor, error),
) (*DevStack, error) {

	nodes := []*DevStackNode{}

	for i := 0; i < count; i++ {
		log.Debug().Msgf(`Creating Node #%d`, i)

		ipfsConnectAddress := ""

		if i > 0 {
			// connect the libp2p scheduler node
			firstNode := nodes[0]
			ipfsConnectAddress = firstNode.IpfsNode.Address()
		}

		// create some random ports to allocate to our servers
		libp2pPort, err := freeport.GetFreePort()
		if err != nil {
			return nil, err
		}

		jsonRpcPort, err := freeport.GetFreePort()
		if err != nil {
			return nil, err
		}

		// construct the ipfs, scheduler, requester, compute and jsonRpc nodes
		ipfsNode, err := ipfs_devstack.NewDevServer(ctx)
		if err != nil {
			return nil, err
		}

		schedulerNode, err := libp2p.NewLibp2pScheduler(ctx, libp2pPort)
		if err != nil {
			return nil, err
		}

		requesterNode, err := requestor_node.NewRequesterNode(ctx, schedulerNode)
		if err != nil {
			return nil, err
		}

		executors, err := getExecutors(ipfsNode.Address())
		if err != nil {
			return nil, err
		}

		computeNode, err := compute_node.NewComputeNode(ctx, schedulerNode, executors)
		if err != nil {
			return nil, err
		}

		jsonRpcNode := jsonrpc.NewBacalhauJsonRpcServer(
			ctx,
			"0.0.0.0",
			jsonRpcPort,
			requesterNode,
		)
		if err != nil {
			return nil, err
		}

		// start the various servers
		err = schedulerNode.Start()
		if err != nil {
			return nil, err
		}

		err = ipfsNode.Start(ipfsConnectAddress)
		if err != nil {
			return nil, err
		}

		err = jsonrpc.StartBacalhauJsonRpcServer(jsonRpcNode)
		if err != nil {
			return nil, err
		}

		// connect subsequent servers to the first one
		if len(nodes) > 0 {
			// connect the libp2p scheduler node
			firstNode := nodes[0]

			// get the libp2p id of the first scheduler node
			libp2pHostId, err := firstNode.SchedulerNode.HostId()
			if err != nil {
				return nil, err
			}

			// connect this scheduler to the first
			firstSchedulerAddress := fmt.Sprintf("/ip4/127.0.0.1/tcp/%d/p2p/%s", firstNode.SchedulerNode.Port, libp2pHostId)
			log.Debug().Msgf("Connect to first libp2p scheduler node: %s", firstSchedulerAddress)
			err = schedulerNode.Connect(firstSchedulerAddress)
			if err != nil {
				return nil, err
			}
		}

		devStackNode := &DevStackNode{
			Ctx:           ctx,
			ComputeNode:   computeNode,
			RequesterNode: requesterNode,
			IpfsNode:      ipfsNode,
			IpfsCli:       ipfs_cli.NewIPFSCli(ipfsNode.Repo),
			SchedulerNode: schedulerNode,
			JSONRpcNode:   jsonRpcNode,
		}

		nodes = append(nodes, devStackNode)
	}

	stack := &DevStack{
		Nodes: nodes,
	}

	return stack, nil
}

func (stack *DevStack) PrintNodeInfo() {

	logString := `
-------------------------------
ipfs
-------------------------------
	`
	for _, node := range stack.Nodes {

		logString = logString + fmt.Sprintf(`
IPFS_PATH=%s ipfs id`, node.IpfsNode.Repo)

	}

	logString += `

-------------------------------
jsonrpc
-------------------------------
	`

	for _, node := range stack.Nodes {

		logString = logString + fmt.Sprintf(`
go run . --jsonrpc-port=%d list`, node.JSONRpcNode.Port)

	}

	log.Info().Msg(logString + "\n")
}
