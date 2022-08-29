package ipfs

import (
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/devstack"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
)

// TODO @enricorotundo #493: duplicate?
// TODO @enricorotundo #493: move next to SetupTestDockerIpfs ?
func SetupTest(t *testing.T, nodes int) (*devstack.DevStackIPFS, *system.CleanupManager) {
	cm := system.NewCleanupManager()
	stack, err := devstack.NewDevStackIPFS(cm, ctx, nodes)
	require.NoError(t, err)

	return stack, cm
}

func TeardownTest(stack *devstack.DevStackIPFS, cm *system.CleanupManager) {
	stack.PrintNodeInfo()
	cm.Cleanup()
}
