package publicapi

import (
	"context"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/types"
	"github.com/stretchr/testify/assert"
)

func TestGet(t *testing.T) {
	c := SetupTests(t)
	ctx := context.Background()

	// Submit a few random jobs to the node:
	var err error
	var job *types.Job
	for i := 0; i < 5; i++ {
		spec, deal := makeJob()
		job, err = c.Submit(ctx, spec, deal)
		assert.NoError(t, err)
	}

	// Should be able to look up one of them:
	job2, ok, err := c.Get(ctx, job.Id)
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, job2.Id, job.Id)
}
