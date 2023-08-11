package ipfs

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bacalhau-project/bacalhau/pkg/devstack"
	"github.com/bacalhau-project/bacalhau/pkg/system"
)

func SetupTest(ctx context.Context, t *testing.T, nodes int) (*devstack.DevStackIPFS, *system.CleanupManager) {
	cm := system.NewCleanupManager()
	stack, err := devstack.NewDevStackIPFS(ctx, cm, nodes)
	require.NoError(t, err)
	return stack, cm
}

func TeardownTest(cm *system.CleanupManager) {
	cm.Cleanup(context.Background())
}
