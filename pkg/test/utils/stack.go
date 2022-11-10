// Package testutils collects common test utilities.
// Functions here create test stacks meant for integration tests
package testutils

import (
	"context"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/localdb/inmemory"

	"github.com/filecoin-project/bacalhau/pkg/computenode"
	devstack "github.com/filecoin-project/bacalhau/pkg/devstack"
	noop_executor "github.com/filecoin-project/bacalhau/pkg/executor/noop"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/node"
	"github.com/filecoin-project/bacalhau/pkg/requesternode"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/transport/inprocess"
	"github.com/stretchr/testify/require"
)

// TODO: Evaluate if TestStack type is needed. It seems to wrap a single node, which already
// exposes an IPFS client, and bunch of other IPFS clients.
type TestStack struct {
	Node      *node.Node
	IpfsStack *devstack.DevStackIPFS
}

type TestStackMultinode struct {
	CleanupManager *system.CleanupManager
	Nodes          []*node.Node
	IpfsStack      *devstack.DevStackIPFS
}

// Docker IPFS stack is designed to be a "as real as possible" stack to write tests against
// but without a libp2p transport - it's useful for testing storage drivers or executors
// it uses:
// * a cluster of real IPFS nodes that form an isolated network
// * you can use the IpfsStack.Add{File,Folder,Text}ToNodes functions to add content and get CIDs
// * in process transport
// * in memory datastore
// * "standard" storage providers - i.e. the default storage stack as used by devstack
// * "standard" executors - i.e. the default executor stack as used by devstack
// * noop verifiers - don't use this stack if you are testing verification
// * IPFS publishers - using the same IPFS cluster as the storage driver
// TODO: this function lies - it only ever returns a single node
func NewDevStackMultiNode(
	ctx context.Context,
	t *testing.T,
	computeNodeConfig computenode.ComputeNodeConfig, //nolint:gocritic
	nodes int,
) *TestStack {
	cm := system.NewCleanupManager()

	ipfsStack, err := devstack.NewDevStackIPFS(ctx, cm, nodes)
	require.NoError(t, err)

	datastore, err := inmemory.NewInMemoryDatastore()
	require.NoError(t, err)

	transport, err := inprocess.NewInprocessTransport()
	require.NoError(t, err)

	nodeConfig := node.NodeConfig{
		IPFSClient:          ipfsStack.IPFSClients[0],
		CleanupManager:      cm,
		LocalDB:             datastore,
		Transport:           transport,
		ComputeNodeConfig:   computeNodeConfig,
		RequesterNodeConfig: requesternode.NewDefaultRequesterNodeConfig(),
	}

	injector := node.NewStandardNodeDependencyInjector()
	injector.VerifiersFactory = devstack.NewNoopVerifiersFactory()

	node, err := node.NewNode(ctx, nodeConfig, injector)
	require.NoError(t, err)

	return &TestStack{
		Node:      node,
		IpfsStack: ipfsStack,
	}
}

// Setup a docker ipfs devstack to run compute node tests against
// This is a shortcut to NewDockerIpfsStackMultiNode but with 1 node
// (formerly SetupTestDockerIpfs)
func NewDevStack(
	ctx context.Context,
	t *testing.T,
	config computenode.ComputeNodeConfig, //nolint:gocritic
) *TestStack {
	return NewDevStackMultiNode(ctx, t, config, 1)
}

// Noop stack is designed to be a "as mocked as possible" stack to write tests against
// it's useful for testing the bidding workflow between requester node and compute nodes
// if you are writing a test that is concerned with "did this control loop emit this event"
// then this is the stack for you (as opposed to "did IPFS actually save the data" in which case
// you want NewDockerIpfsStackMultiNode)
// it uses:
// * in process transport
// * in memory datastore
// * noop storage providers
// * noop executors
// * noop verifiers
// * noop publishers
func NewNoopStack(
	ctx context.Context,
	t *testing.T,
	computeNodeconfig computenode.ComputeNodeConfig,
	noopExecutorConfig noop_executor.ExecutorConfig,
) *TestStack {
	cm := system.NewCleanupManager()

	datastore, err := inmemory.NewInMemoryDatastore()
	require.NoError(t, err)

	transport, err := inprocess.NewInprocessTransport()
	require.NoError(t, err)

	nodeConfig := node.NodeConfig{
		CleanupManager:      cm,
		LocalDB:             datastore,
		Transport:           transport,
		ComputeNodeConfig:   computeNodeconfig,
		RequesterNodeConfig: requesternode.NewDefaultRequesterNodeConfig(),
	}

	injector := devstack.NewNoopNodeDependencyInjector()
	injector.ExecutorsFactory = devstack.NewNoopExecutorsFactoryWithConfig(noopExecutorConfig)

	node, err := node.NewNode(ctx, nodeConfig, injector)
	require.NoError(t, err)

	err = transport.Start(ctx)
	require.NoError(t, err)

	return &TestStack{
		Node: node,
	}
}

// same as n
func NewNoopStackMultinode(
	ctx context.Context,
	t *testing.T,
	count int,
	computeNodeconfig computenode.ComputeNodeConfig,
	noopExecutorConfig noop_executor.ExecutorConfig,
	inprocessTransportConfig inprocess.InProcessTransportClusterConfig,
) *TestStackMultinode {
	cm := system.NewCleanupManager()

	nodes := []*node.Node{}

	inprocessTransportConfig.Count = count
	cluster, err := inprocess.NewInProcessTransportCluster(inprocessTransportConfig)
	require.NoError(t, err)

	for i := 0; i < count; i++ {
		datastore, err := inmemory.NewInMemoryDatastore()
		require.NoError(t, err)

		transport := cluster.GetTransport(i)
		nodeConfig := node.NodeConfig{
			CleanupManager:      cm,
			LocalDB:             datastore,
			Transport:           transport,
			ComputeNodeConfig:   computeNodeconfig,
			RequesterNodeConfig: requesternode.NewDefaultRequesterNodeConfig(),
		}

		injector := devstack.NewNoopNodeDependencyInjector()
		injector.ExecutorsFactory = devstack.NewNoopExecutorsFactoryWithConfig(noopExecutorConfig)
		node, err := node.NewNode(ctx, nodeConfig, injector)
		require.NoError(t, err)
		err = transport.Start(ctx)
		require.NoError(t, err)

		nodes = append(nodes, node)
	}

	return &TestStackMultinode{
		Nodes:          nodes,
		CleanupManager: cm,
	}
}
