package verifier

import (
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/devstack"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/stretchr/testify/assert"
)

func SetupTest(t *testing.T, nodes int) (
	*devstack.DevStack_IPFS, *system.CleanupManager) {

	cm := system.NewCleanupManager()
	stack, err := devstack.NewDevStack_IPFS(cm, nodes)
	assert.NoError(t, err, "unable to create devstack")

	return stack, cm
}

func TeardownTest(stack *devstack.DevStack_IPFS, cm *system.CleanupManager) {
	stack.PrintNodeInfo()
	cm.Cleanup()
}
