package publicapi

import (
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/stretchr/testify/assert"
)

func TestGet(t *testing.T) {
	ctx, c := SetupTests(t)
	defer system.CleanupTracer()

	ctx, span := system.Span(ctx, "publicapi/client_test", "TestGet")
	defer span.End()

	// Submit a few random jobs to the node:
	var err error
	var job *executor.Job
	for i := 0; i < 5; i++ {
		spec, deal := MakeGenericJob()
		job, err = c.Submit(ctx, spec, deal)
		assert.NoError(t, err)
	}

	// Should be able to look up one of them:
	job2, ok, err := c.Get(ctx, job.Id)
	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, job2.Id, job.Id)
}
