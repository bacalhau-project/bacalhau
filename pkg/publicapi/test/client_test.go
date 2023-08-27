//go:build unit || !integration

package test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	testutils "github.com/bacalhau-project/bacalhau/pkg/test/utils"
)

func TestGet(t *testing.T) {
	logger.ConfigureTestLogging(t)
	n, c := setupNodeForTest(t)
	defer n.CleanupManager.Cleanup(context.Background())

	ctx := context.Background()

	// Submit a few random jobs to the node:
	var err error
	var j *model.Job
	for i := 0; i < 5; i++ {
		genericJob := testutils.MakeNoopJob(t)
		j, err = c.Submit(ctx, genericJob)
		require.NoError(t, err)
	}

	// Should be able to look up one of them:
	job2, ok, err := c.Get(ctx, j.Metadata.ID)
	require.NoError(t, err)
	require.True(t, ok)
	require.Equal(t, job2.Job.Metadata.ID, j.Metadata.ID)
}
