package test

import (
	"testing"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/compute_node"
	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/executor/noop"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/requestor_node"
	"github.com/filecoin-project/bacalhau/pkg/scheduler/inprocess"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/types"
	"github.com/stretchr/testify/assert"
)

func TestSchedulerSanity(t *testing.T) {
	ctx, _ := system.GetCancelContext()
	executors := map[string]executor.Executor{}
	scheduler, err := inprocess.NewInprocessScheduler(ctx)
	assert.NoError(t, err)
	_, err = compute_node.NewComputeNode(ctx, scheduler, executors)
	assert.NoError(t, err)
	_, err = requestor_node.NewRequesterNode(ctx, scheduler)
	assert.NoError(t, err)
}

func TestSchedulerSubmitJob(t *testing.T) {
	ctx, _ := system.GetCancelContext()
	noopExecutor, err := noop.NewNoopExecutor()
	assert.NoError(t, err)
	executors := map[string]executor.Executor{
		"noop": noopExecutor,
	}
	scheduler, err := inprocess.NewInprocessScheduler(ctx)
	assert.NoError(t, err)
	_, err = compute_node.NewComputeNode(ctx, scheduler, executors)
	assert.NoError(t, err)
	_, err = requestor_node.NewRequesterNode(ctx, scheduler)
	assert.NoError(t, err)

	// first let's test submitting a job with an engine we do not have
	spec := &types.JobSpec{
		Image:      "image",
		Engine:     "apples",
		Entrypoint: "entrypoint",
		Env:        []string{"env"},
		Inputs: []types.StorageSpec{
			{
				Engine: "ipfs",
			},
		},
	}

	deal := &types.JobDeal{
		Concurrency: 1,
	}

	_, err = scheduler.SubmitJob(spec, deal)
	assert.NoError(t, err)
	time.Sleep(time.Second * 1)
	assert.Equal(t, 0, len(noopExecutor.Jobs))
	spec.Engine = "noop"
	jobSelected, err := scheduler.SubmitJob(spec, deal)
	assert.NoError(t, err)
	time.Sleep(time.Second * 1)
	assert.Equal(t, 1, len(noopExecutor.Jobs))
	assert.Equal(t, jobSelected.Id, noopExecutor.Jobs[0].Id)
}
