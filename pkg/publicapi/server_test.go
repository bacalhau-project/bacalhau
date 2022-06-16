package publicapi

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestList(t *testing.T) {
	c := SetupTests(t)

	// Should have no jobs initially:
	jobs, err := c.List()
	assert.NoError(t, err)
	assert.Empty(t, jobs)

	// Submit a random job to the node:
	_, err = c.Submit(MakeGenericJob())
	assert.NoError(t, err)

	// Should now have one job:
	jobs, err = c.List()
	assert.NoError(t, err)
	assert.Len(t, jobs, 1)
}
