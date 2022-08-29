package verifier

import (
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/devstack"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/stretchr/testify/require"
)

// TODO @enricorotundo #493: duplicate?
// TODO @enricorotundo #493: move next to SetupTestDockerIpfs ?
func SetupTest(t *testing.T, nodes int) (*devstack.DevStackIPFS, *system.CleanupManager) {
	cm := system.NewCleanupManager()
	stack, err := devstack.NewDevStackIPFS(cm, nodes)
	require.NoError(t, err, "unable to create devstack")

	return stack, cm
}

func TeardownTest(stack *devstack.DevStackIPFS, cm *system.CleanupManager) {
	stack.PrintNodeInfo()
	cm.Cleanup()
}
