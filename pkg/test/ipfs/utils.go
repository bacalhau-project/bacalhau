package ipfs

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bacalhau-project/bacalhau/pkg/devstack"
	"github.com/bacalhau-project/bacalhau/pkg/system/cleanup"
)

func SetupTest(ctx context.Context, t *testing.T, nodes int) (*devstack.DevStackIPFS, *cleanup.CleanupManager) {
	cm := cleanup.NewCleanupManager()
	stack, err := devstack.NewDevStackIPFS(ctx, cm, nodes)
	require.NoError(t, err)
	return stack, cm
}

func TeardownTest(cm *cleanup.CleanupManager) {
	cm.Cleanup(context.Background())
}
