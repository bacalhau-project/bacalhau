package run

import (
	"context"
	"fmt"

	"github.com/testground/sdk-go/network"
	"github.com/testground/sdk-go/runtime"
	"github.com/testground/sdk-go/sync"
)

const (
	StateInitializedGlobal   = sync.State("initialized_global")
	StateInitializedGroupFmt = "initialized_group_%s"
)

// InitSyncClientFactory is the function that will be called to initialize a
// sync client as part of an InitContext.
//
// Replaced in testing.
var InitSyncClientFactory = func(ctx context.Context, env *runtime.RunEnv) sync.Client {
	// cannot assign sync.MustBoundClient directly because Go can't infer the contravariance
	// in the return type (i.e. that *sync.DefaultClient satisfies sync.Client).
	return sync.MustBoundClient(ctx, env)
}

// InitContext encapsulates a sync client, a net client, and global and
// group-scoped seq numbers assigned to this test instance by the sync service.
//
// The states we signal to acquire the global and group-scoped seq numbers are:
//  - initialized_global
//  - initialized_group_<id>
type InitContext struct {
	SyncClient sync.Client
	NetClient  *network.Client
	GlobalSeq  int64
	GroupSeq   int64

	runenv *runtime.RunEnv
}

// init can be safely invoked on a nil reference.
func (ic *InitContext) init(runenv *runtime.RunEnv) {
	var (
		grpstate  = sync.State(fmt.Sprintf(StateInitializedGroupFmt, runenv.TestGroupID))
		client    = InitSyncClientFactory(context.Background(), runenv)
		netclient = network.NewClient(client, runenv)
	)

	runenv.RecordMessage("waiting for network to initialize")
	netclient.MustWaitNetworkInitialized(context.Background())
	runenv.RecordMessage("network initialization complete")

	*ic = InitContext{
		SyncClient: client,
		NetClient:  netclient,
		GlobalSeq:  client.MustSignalEntry(context.Background(), StateInitializedGlobal),
		GroupSeq:   client.MustSignalEntry(context.Background(), grpstate),
		runenv:     runenv,
	}

	runenv.AttachSyncClient(client)

	runenv.RecordMessage("claimed sequence numbers; global=%d, group(%s)=%d", ic.GlobalSeq, runenv.TestGroupID, ic.GroupSeq)
}

func (ic *InitContext) close() {
	if err := ic.SyncClient.Close(); err != nil {
		panic(err)
	}
}

// WaitAllInstancesInitialized waits for all instances to initialize.
func (ic *InitContext) WaitAllInstancesInitialized(ctx context.Context) error {
	return <-ic.SyncClient.MustBarrier(ctx, StateInitializedGlobal, ic.runenv.TestInstanceCount).C
}

// MustWaitAllInstancesInitialized calls WaitAllInstancesInitialized, and
// panics if it errors.
func (ic *InitContext) MustWaitAllInstancesInitialized(ctx context.Context) {
	if err := ic.WaitAllInstancesInitialized(ctx); err != nil {
		panic(err)
	}
}

// WaitGroupInstancesInitialized waits for all group instances to initialize.
func (ic *InitContext) WaitGroupInstancesInitialized(ctx context.Context) error {
	grpstate := sync.State(fmt.Sprintf(StateInitializedGroupFmt, ic.runenv.TestGroupID))
	return <-ic.SyncClient.MustBarrier(ctx, grpstate, ic.runenv.TestGroupInstanceCount).C
}

// MustWaitGroupInstancesInitialized calls WaitGroupInstancesInitialized, and
// panics if it errors.
func (ic *InitContext) MustWaitGroupInstancesInitialized(ctx context.Context) {
	if err := ic.WaitGroupInstancesInitialized(ctx); err != nil {
		panic(err)
	}
}
