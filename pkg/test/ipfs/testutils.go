package ipfs

import (
	"context"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/devstack"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/stretchr/testify/assert"
)

func SetupTest(t *testing.T, nodes int) (
	*devstack.DevStack_IPFS, context.Context, context.CancelFunc) {

	ctx, cancel := system.WithSignalShutdown(context.Background())
	stack, err := devstack.NewDevStack_IPFS(ctx, nodes)
	assert.NoError(t, err)

	return stack, ctx, cancel
}

func TeardownTest(stack *devstack.DevStack_IPFS, cancel context.CancelFunc) {
	stack.PrintNodeInfo()
	cancel()
}
