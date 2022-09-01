// Package testutils collects common test utilities.
// Functions here create test stacks meant for integration tests
package testutils

import (
	"context"
	"fmt"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/computenode"
	"github.com/filecoin-project/bacalhau/pkg/controller"
	devstack "github.com/filecoin-project/bacalhau/pkg/devstack"
	"github.com/filecoin-project/bacalhau/pkg/executor"
	noop_executor "github.com/filecoin-project/bacalhau/pkg/executor/noop"
	executor_util "github.com/filecoin-project/bacalhau/pkg/executor/util"
	"github.com/filecoin-project/bacalhau/pkg/localdb/inmemory"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/model"
	publisher_util "github.com/filecoin-project/bacalhau/pkg/publisher/util"
	"github.com/filecoin-project/bacalhau/pkg/requesternode"
	"github.com/filecoin-project/bacalhau/pkg/storage/noop"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/transport/inprocess"
	verifier_util "github.com/filecoin-project/bacalhau/pkg/verifier/util"
	"github.com/stretchr/testify/require"
)

type TestStack struct {
	ComputeNode    *computenode.ComputeNode
	RequestorNode  *requesternode.RequesterNode
	Controller     *controller.Controller
	CleanupManager *system.CleanupManager
	IpfsStack      *devstack.DevStackIPFS
	Executors      map[model.EngineType]executor.Executor
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
func NewDockerIpfsStackMultiNode(
	ctx context.Context,
	t *testing.T,
	config computenode.ComputeNodeConfig, //nolint:gocritic
	nodes int,
) *TestStack {
	cm := system.NewCleanupManager()

	ipfsStack, err := devstack.NewDevStackIPFS(ctx, cm, nodes)
	require.NoError(t, err)

	apiAddress := ipfsStack.Nodes[0].IpfsClient.APIAddress()
	transport, err := inprocess.NewInprocessTransport()
	require.NoError(t, err)

	datastore, err := inmemory.NewInMemoryDatastore()
	require.NoError(t, err)

	ipfsID := ipfsStack.Nodes[0].IpfsNode.ID()

	storageProviders, err := executor_util.NewStandardStorageProviders(ctx, cm, executor_util.StandardStorageProviderOptions{
		IPFSMultiaddress: apiAddress,
	})
	require.NoError(t, err)
	executors, err := executor_util.NewStandardExecutors(
		ctx,
		cm,
		executor_util.StandardExecutorOptions{
			DockerID: fmt.Sprintf("devstacknode0-%s", ipfsID),
			Storage: executor_util.StandardStorageProviderOptions{
				IPFSMultiaddress: apiAddress,
			},
		},
	)
	require.NoError(t, err)

	ctrl, err := controller.NewController(ctx, cm, datastore, transport, storageProviders)
	require.NoError(t, err)

	verifiers, err := verifier_util.NewNoopVerifiers(
		ctx,
		cm,
		ctrl.GetStateResolver(),
	)
	require.NoError(t, err)

	publishers, err := publisher_util.NewIPFSPublishers(
		ctx,
		cm,
		ctrl.GetStateResolver(),
		apiAddress,
	)
	require.NoError(t, err)

	computeNode, err := computenode.NewComputeNode(
		ctx,
		cm,
		ctrl,
		executors,
		verifiers,
		publishers,
		config,
	)
	require.NoError(t, err)

	return &TestStack{
		ComputeNode:    computeNode,
		IpfsStack:      ipfsStack,
		Controller:     ctrl,
		CleanupManager: cm,
		Executors:      executors,
	}
}

// Setup a docker ipfs devstack to run compute node tests against
// This is a shortcut to NewDockerIpfsStackMultiNode but with 1 node
// (formerly SetupTestDockerIpfs)
func NewDockerIpfsStack(
	ctx context.Context,
	t *testing.T,
	config computenode.ComputeNodeConfig, //nolint:gocritic
) *TestStack {
	return NewDockerIpfsStackMultiNode(ctx, t, config, 1)
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

	transport, err := inprocess.NewInprocessTransport()
	require.NoError(t, err)

	datastore, err := inmemory.NewInMemoryDatastore()
	require.NoError(t, err)

	executors, err := executor_util.NewNoopExecutors(ctx, cm, noopExecutorConfig)
	require.NoError(t, err)

	storageProviders, err := executor_util.NewNoopStorageProviders(ctx, cm, noop.StorageConfig{})
	require.NoError(t, err)

	ctrl, err := controller.NewController(ctx, cm, datastore, transport, storageProviders)
	require.NoError(t, err)

	verifiers, err := verifier_util.NewNoopVerifiers(ctx, cm, ctrl.GetStateResolver())
	require.NoError(t, err)

	publishers, err := publisher_util.NewNoopPublishers(ctx, cm, ctrl.GetStateResolver())
	require.NoError(t, err)

	requestorNode, err := requesternode.NewRequesterNode(
		ctx,
		cm,
		ctrl,
		verifiers,
		requesternode.RequesterNodeConfig{},
	)
	require.NoError(t, err)

	computeNode, err := computenode.NewComputeNode(
		ctx,
		cm,
		ctrl,
		executors,
		verifiers,
		publishers,
		computeNodeconfig,
	)
	if err != nil {
		t.Fatal(err)
	}

	err = ctrl.Start(ctx)
	if err != nil {
		t.Fatal(err)
	}

	if err = transport.Start(ctx); err != nil {
		t.Fatal(err)
	}

	return &TestStack{
		ComputeNode:    computeNode,
		RequestorNode:  requestorNode,
		Controller:     ctrl,
		CleanupManager: cm,
	}
}
