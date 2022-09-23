package utils

import (
	"context"

	"github.com/filecoin-project/bacalhau/pkg/node"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/testground/sdk-go/run"
	"github.com/testground/sdk-go/runtime"
	"github.com/testground/sdk-go/sync"
)

// ExecuteTestFn is the function that will be executed by the test cases.
type ExecuteTestFn func(ctx context.Context, runenv *runtime.RunEnv, initCtx *run.InitContext, node *node.Node) error

// ExecuteTest executes the test cases.
// This method will create the nodes, execute the test cases and release the network.
func ExecuteTest(ctx context.Context, runenv *runtime.RunEnv, initCtx *run.InitContext, execute ExecuteTestFn) error {
	// Create the node.
	node, err := bootstrap(ctx, runenv, initCtx)
	if err != nil {
		return err
	}
	defer node.CleanupManager.Cleanup()

	if initCtx.GlobalSeq == 1 {
		// ExecuteTestFn the test cases if this is a requester node
		err = execute(ctx, runenv, initCtx, node)
		if err != nil {
			return err
		}
		runenv.RecordSuccess()
	}

	// Release the nodes, or wait for the requester node to release us.
	return releaseOrWait(ctx, runenv, initCtx)
}

// Bootstrap creates a node and waits for the network to be ready.
// The first node created by testground will be act as the bootstrap node for the rest of the nodes.
func bootstrap(ctx context.Context, runenv *runtime.RunEnv, initCtx *run.InitContext) (*node.Node, error) {
	// State that will be used by non-bootstrap nodes to signal that they are ready
	readyState := sync.State("ready")

	initCtx.MustWaitAllInstancesInitialized(ctx)
	seq := initCtx.GlobalSeq

	client := sync.MustBoundClient(ctx, runenv)
	defer client.Close()

	// Topic that will be used to publish the bootstrap node info.
	bootstrapTopic := sync.NewTopic("bootstrap", &NodeInfo{})

	// Bacalhau requires a config file to be present in the system.
	err := system.InitConfigForTesting()
	if err != nil {
		return nil, err
	}
	cm := system.NewCleanupManager()

	var newNode *node.Node

	if seq == 1 {
		// This will be our bootstrap/leader node.
		newNode, err = CreateAndStartNode(ctx, cm, &NodeInfo{})
		if err != nil {
			return nil, err
		}

		var nodeInfo *NodeInfo
		nodeInfo, err = GetNodeInfo(ctx, newNode)
		if err != nil {
			return nil, err
		}

		// Publish bootstrap node info
		client.MustPublish(ctx, bootstrapTopic, nodeInfo)
		runenv.RecordMessage("published bootstrap node info. %v", nodeInfo)

		// let's wait for the followers to signal.
		runenv.RecordMessage("waiting for %d instances to become ready", runenv.TestInstanceCount)
		err = <-client.MustBarrier(ctx, readyState, runenv.TestInstanceCount-1).C
		if err != nil {
			return nil, err
		}
	} else {
		// This will be a follower node.
		peerChannel := make(chan *NodeInfo)

		// Subscribe to the bootstrap topic to get the bootstrap node info.
		subscription := client.MustSubscribe(ctx, bootstrapTopic, peerChannel)
		nodeInfo := <-peerChannel
		runenv.RecordMessage("received bootstrap peer info %v", nodeInfo)
		subscription.Done()

		// Create and start the node using the bootstrap node info.
		newNode, err = CreateAndStartNode(ctx, cm, nodeInfo)
		if err != nil {
			return nil, err
		}

		// signal entry in the 'ready' state.
		client.MustSignalEntry(ctx, readyState)
	}

	return newNode, nil
}

// Blocks the nodes until the test suite is completed and released by the requester node.
// If this is a follower node, then it will wait for the release signal from the requester node.
// If this is a requester node, then it should call this method onces all the test cases have been executed.
func releaseOrWait(ctx context.Context, runenv *runtime.RunEnv, initCtx *run.InitContext) error {
	// State that will be used by the requester node to signal that the test suite is completed.
	releasedState := sync.State("released")

	client := sync.MustBoundClient(ctx, runenv)
	defer client.Close()

	seq := initCtx.GlobalSeq

	if seq == 1 {
		// signal on the 'released' state.
		client.MustSignalEntry(ctx, releasedState)
		runenv.RecordMessage("release signal sent")
	} else {
		// wait until the leader releases us.
		err := <-client.MustBarrier(ctx, releasedState, 1).C
		if err != nil {
			return err
		}
		runenv.RecordMessage("instance %d released", seq)
	}
	return nil
}
