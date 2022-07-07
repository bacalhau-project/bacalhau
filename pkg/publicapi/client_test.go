package publicapi

import (
	"context"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGet(t *testing.T) {
	t.Skip("TEMP_SKIP_FOR_NULL_POINTER_FAST_FAIL_TEST")

	c, cm := SetupTests(t)
	defer cm.Cleanup()

	ctx, span := system.Span(context.Background(),
		"publicapi/client_test", "TestGet")
	defer span.End()

	// Submit a few random jobs to the node:
	var err error
	var job *executor.Job
	for i := 0; i < 5; i++ {
		spec, deal := MakeGenericJob()
		job, err = c.Submit(ctx, spec, deal, nil)
		require.NoError(t, err)
	}

	// Should be able to look up one of them:
	job2, ok, err := c.Get(ctx, job.ID)
	require.NoError(t, err)
	assert.True(t, ok)
	require.Equal(t, job2.ID, job.ID)
}
