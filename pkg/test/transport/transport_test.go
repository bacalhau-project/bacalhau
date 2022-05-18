package transport_test

import (
	"reflect"
	"testing"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/compute_node"
	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/executor/noop"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/requestor_node"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/transport/inprocess"
	"github.com/filecoin-project/bacalhau/pkg/types"
	"github.com/stretchr/testify/assert"
)

func TestTransportSanity(t *testing.T) {
	executors := map[string]executor.Executor{}
	transport, err := inprocess.NewInprocessTransport()
	assert.NoError(t, err)
	_, err = compute_node.NewComputeNode(transport, executors)
	assert.NoError(t, err)
	_, err = requestor_node.NewRequesterNode(transport)
	assert.NoError(t, err)
}

func TestSchedulerSubmitJob(t *testing.T) {
	noopExecutor, err := noop.NewNoopExecutor()
	assert.NoError(t, err)
	executors := map[string]executor.Executor{
		"noop": noopExecutor,
	}
	transport, err := inprocess.NewInprocessTransport()
	assert.NoError(t, err)
	_, err = compute_node.NewComputeNode(transport, executors)
	assert.NoError(t, err)
	_, err = requestor_node.NewRequesterNode(transport)
	assert.NoError(t, err)

	// first let's test submitting a job with an engine we do not have
	spec := &types.JobSpec{
		Engine: "apples",
		Vm: types.JobSpecVm{
			Image:      "image",
			Entrypoint: []string{"entrypoint"},
			Env:        []string{"env"},
		},
		Inputs: []types.StorageSpec{
			{
				Engine: "ipfs",
			},
		},
	}

	deal := &types.JobDeal{
		Concurrency: 1,
	}

	_, err = transport.SubmitJob(spec, deal)
	assert.NoError(t, err)
	time.Sleep(time.Second * 1)
	assert.Equal(t, 0, len(noopExecutor.Jobs))
	spec.Engine = "noop"
	jobSelected, err := transport.SubmitJob(spec, deal)
	assert.NoError(t, err)
	time.Sleep(time.Second * 1)
	assert.Equal(t, 1, len(noopExecutor.Jobs))
	assert.Equal(t, jobSelected.Id, noopExecutor.Jobs[0].Id)
}

func TestTransportEvents(t *testing.T) {
	noopExecutor, err := noop.NewNoopExecutor()
	assert.NoError(t, err)
	executors := map[string]executor.Executor{
		"noop": noopExecutor,
	}
	transport, err := inprocess.NewInprocessTransport()
	assert.NoError(t, err)
	_, err = compute_node.NewComputeNode(transport, executors)
	assert.NoError(t, err)
	_, err = requestor_node.NewRequesterNode(transport)
	assert.NoError(t, err)

	// first let's test submitting a job with an engine we do not have
	spec := &types.JobSpec{
		Engine: "noop",
		Vm: types.JobSpecVm{
			Image:      "image",
			Entrypoint: []string{"entrypoint"},
			Env:        []string{"env"},
		},
		Inputs: []types.StorageSpec{
			{
				Engine: "ipfs",
			},
		},
	}

	deal := &types.JobDeal{
		Concurrency: 1,
	}

	_, err = transport.SubmitJob(spec, deal)
	assert.NoError(t, err)
	time.Sleep(time.Second * 1)

	expectedEventNames := []string{
		system.JOB_EVENT_CREATED,
		system.JOB_EVENT_BID,
		system.JOB_EVENT_BID_ACCEPTED,
		system.JOB_EVENT_RESULTS,
	}
	actualEventNames := []string{}

	for _, event := range transport.Events {
		actualEventNames = append(actualEventNames, event.EventName)
	}

	assert.True(t, reflect.DeepEqual(expectedEventNames, actualEventNames), "event list is correct")
}
