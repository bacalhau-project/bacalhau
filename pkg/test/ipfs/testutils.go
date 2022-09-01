package ipfs

import (
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/devstack"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/stretchr/testify/require"
)

func SetupTest(t *testing.T, nodes int) (*devstack.DevStackIPFS, *system.CleanupManager) {
	cm := system.NewCleanupManager()
	cm.RegisterCallback(system.CleanupTraceProvider)
	stack, err := devstack.NewDevStackIPFS(cm, nodes)
	require.NoError(t, err)

	return stack, cm
}

func TeardownTest(stack *devstack.DevStackIPFS, cm *system.CleanupManager) {
	stack.PrintNodeInfo()
	cm.Cleanup()
}
