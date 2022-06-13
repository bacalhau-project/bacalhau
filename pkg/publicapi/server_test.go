package publicapi

import (
	"context"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/types"
	"github.com/filecoin-project/bacalhau/pkg/verifier"
	"github.com/stretchr/testify/assert"
)

func TestList(t *testing.T) {
	c := SetupTests(t)
	defer system.CleanupTracer()

	ctx, span := system.Span(context.Background(), "publicapi/server_test",
		"TestList")
	defer span.End()

	// Should have no jobs initially:
	jobs, err := c.List(ctx)
	assert.NoError(t, err)
	assert.Empty(t, jobs)

	// Submit a random job to the node:
	spec, deal := makeJob()
	_, err = c.Submit(ctx, spec, deal)
	assert.NoError(t, err)

	// Should now have one job:
	jobs, err = c.List(ctx)
	assert.NoError(t, err)
	assert.Len(t, jobs, 1)
}

func makeJob() (*types.JobSpec, *types.JobDeal) {
	jobSpec := types.JobSpec{
		Engine:   string(executor.EXECUTOR_DOCKER),
		Verifier: string(verifier.VERIFIER_IPFS),
		Vm: types.JobSpecVm{
			Image: "ubuntu:latest",
			Entrypoint: []string{
				"cat",
				"/data/file.txt",
			},
		},
	}

	jobDeal := types.JobDeal{
		Concurrency: 1,
	}

	return &jobSpec, &jobDeal
}
