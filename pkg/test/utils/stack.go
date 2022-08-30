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
	publisher_util "github.com/filecoin-project/bacalhau/pkg/publisher/util"
	"github.com/filecoin-project/bacalhau/pkg/requesternode"
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
	Executors      map[executor.EngineType]executor.Executor
}

// Setup a docker ipfs devstack to run compute node tests against (formerly SetupTestDockerIpfs)
func NewDockerIpfsStack(
	t *testing.T,
	config computenode.ComputeNodeConfig, //nolint:gocritic
) *TestStack {
	cm := system.NewCleanupManager()

	ipfsStack, err := devstack.NewDevStackIPFS(cm, 1)
	require.NoError(t, err)

	apiAddress := ipfsStack.Nodes[0].IpfsClient.APIAddress()
	transport, err := inprocess.NewInprocessTransport()
	require.NoError(t, err)

	datastore, err := inmemory.NewInMemoryDatastore()
	require.NoError(t, err)

	ipfsID := ipfsStack.Nodes[0].IpfsNode.ID()

	storageProviders, err := executor_util.NewStandardStorageProviders(cm, executor_util.StandardStorageProviderOptions{
		IPFSMultiaddress: apiAddress,
	})
	require.NoError(t, err)
	executors, err := executor_util.NewStandardExecutors(
		cm,
		executor_util.StandardExecutorOptions{
			DockerID: fmt.Sprintf("devstacknode0-%s", ipfsID),
			Storage: executor_util.StandardStorageProviderOptions{
				IPFSMultiaddress: apiAddress,
			},
		},
	)
	require.NoError(t, err)

	ctrl, err := controller.NewController(cm, datastore, transport, storageProviders)
	require.NoError(t, err)

	verifiers, err := verifier_util.NewNoopVerifiers(
		cm,
		ctrl.GetStateResolver(),
	)
	require.NoError(t, err)

	publishers, err := publisher_util.NewIPFSPublishers(
		cm,
		ctrl.GetStateResolver(),
		apiAddress,
	)
	require.NoError(t, err)

	computeNode, err := computenode.NewComputeNode(
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

// Setup a full noop stack to run tests against (formerly SetupTestNoop)
func NewNoopStack(
	t *testing.T,
	//nolint:gocritic
	computeNodeconfig computenode.ComputeNodeConfig,
	noopExecutorConfig noop_executor.ExecutorConfig,
) *TestStack {
	cm := system.NewCleanupManager()

	transport, err := inprocess.NewInprocessTransport()
	require.NoError(t, err)

	datastore, err := inmemory.NewInMemoryDatastore()
	require.NoError(t, err)

	executors, err := executor_util.NewNoopExecutors(cm, noopExecutorConfig)
	require.NoError(t, err)

	storageProviders, err := executor_util.NewNoopStorageProviders(cm)
	require.NoError(t, err)

	ctrl, err := controller.NewController(cm, datastore, transport, storageProviders)
	require.NoError(t, err)

	verifiers, err := verifier_util.NewNoopVerifiers(cm, ctrl.GetStateResolver())
	require.NoError(t, err)

	publishers, err := publisher_util.NewNoopPublishers(cm, ctrl.GetStateResolver())
	require.NoError(t, err)

	requestorNode, err := requesternode.NewRequesterNode(
		cm,
		ctrl,
		verifiers,
		requesternode.RequesterNodeConfig{},
	)
	require.NoError(t, err)

	computeNode, err := computenode.NewComputeNode(
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

	ctx := context.Background()
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
