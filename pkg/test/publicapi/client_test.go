//go:build unit || !integration

package publicapi

import (
	"context"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/system"
	testutils "github.com/filecoin-project/bacalhau/pkg/test/utils"
	"github.com/stretchr/testify/require"
)

func TestGet(t *testing.T) {
	logger.ConfigureTestLogging(t)
	n, c := setupNodeForTest(t)
	defer n.CleanupManager.Cleanup()

	ctx, span := system.Span(context.Background(),
		"publicapi/client_test", "TestGet")
	defer span.End()

	// Submit a few random jobs to the node:
	var err error
	var j *model.Job
	for i := 0; i < 5; i++ {
		genericJob := testutils.MakeGenericJob()
		j, err = c.Submit(ctx, genericJob)
		require.NoError(t, err)
	}

	// Should be able to look up one of them:
	job2, ok, err := c.Get(ctx, j.Metadata.ID)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, job2.Metadata.ID, j.Metadata.ID)
}
