package ipfs

import (
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/devstack"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/stretchr/testify/assert"
)

func SetupTest(t *testing.T, nodes int) (*devstack.DevStackIPFS, *system.CleanupManager) {
	cm := system.NewCleanupManager()
	cm.RegisterCallback(system.CleanupTracer)
	stack, err := devstack.NewDevStackIPFS(cm, nodes)
	assert.NoError(t, err)

	return stack, cm
}

func TeardownTest(stack *devstack.DevStackIPFS, cm *system.CleanupManager) {
	stack.PrintNodeInfo()
	cm.Cleanup()
}
