package publicapi

import (
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/types"
	"github.com/stretchr/testify/assert"
)

func TestGet(t *testing.T) {
	c := SetupTests(t)

	// Submit a few random jobs to the node:
	var err error
	var job *types.Job
	for i := 0; i < 5; i++ {
		job, err = c.Submit(makeJob())
		assert.NoError(t, err)
	}

	// Should be able to look up one of them:
	job2, err := c.Get(job.Id)
	assert.NoError(t, err)
	assert.Equal(t, job2.Id, job.Id)
}
