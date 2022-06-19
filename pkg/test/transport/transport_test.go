package transport_test

import (
	"context"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/compute_node"
	"github.com/filecoin-project/bacalhau/pkg/executor"
	executor_noop "github.com/filecoin-project/bacalhau/pkg/executor/noop"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/requestor_node"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/transport/inprocess"
	"github.com/filecoin-project/bacalhau/pkg/types"
	"github.com/filecoin-project/bacalhau/pkg/verifier"
	verifier_noop "github.com/filecoin-project/bacalhau/pkg/verifier/noop"
	"github.com/stretchr/testify/assert"
)

func setupTest(t *testing.T) (
	*inprocess.Transport,
	*executor_noop.Executor,
	*verifier_noop.Verifier,
) {
	noopExecutor, err := executor_noop.NewExecutor()
	assert.NoError(t, err)

	noopVerifier, err := verifier_noop.NewVerifier()
	assert.NoError(t, err)

	executors := map[string]executor.Executor{
		string(executor.EXECUTOR_NOOP): noopExecutor,
	}

	verifiers := map[string]verifier.Verifier{
		string(verifier.VERIFIER_NOOP): noopVerifier,
	}

	transport, err := inprocess.NewInprocessTransport()
	assert.NoError(t, err)

	_, err = compute_node.NewComputeNode(
		transport,
		executors,
		verifiers,
		compute_node.NewDefaultJobSelectionPolicy(),
	)
	assert.NoError(t, err)

	_, err = requestor_node.NewRequesterNode(transport)
	assert.NoError(t, err)

	return transport, noopExecutor, noopVerifier
}

func TestTransportSanity(t *testing.T) {
	executors := map[string]executor.Executor{}
	verifiers := map[string]verifier.Verifier{}
	transport, err := inprocess.NewInprocessTransport()
	assert.NoError(t, err)
	_, err = compute_node.NewComputeNode(
		transport,
		executors,
		verifiers,
		compute_node.NewDefaultJobSelectionPolicy(),
	)
	assert.NoError(t, err)
	_, err = requestor_node.NewRequesterNode(transport)
	assert.NoError(t, err)
}

func TestSchedulerSubmitJob(t *testing.T) {
	ctx := context.Background()
	transport, noopExecutor, _ := setupTest(t)

	// first let's test submitting a job with an engine we do not have
	spec := &types.JobSpec{
		Engine:   "apples",
		Verifier: string(verifier.VERIFIER_NOOP),
		VM: types.JobSpecVm{
			Image:      "image",
			Entrypoint: []string{"entrypoint"},
			Env:        []string{"env"},
		},
		Inputs: []types.StorageSpec{
			{
				Engine: storage.IPFS_DEFAULT,
			},
		},
	}

	deal := &types.JobDeal{
		Concurrency: 1,
	}

	_, err := transport.SubmitJob(ctx, spec, deal)
	assert.NoError(t, err)

	time.Sleep(time.Second * 1)
	assert.Equal(t, 0, len(noopExecutor.Jobs))

	spec.Engine = string(executor.EXECUTOR_NOOP)
	jobSelected, err := transport.SubmitJob(ctx, spec, deal)
	assert.NoError(t, err)

	time.Sleep(time.Second * 1)
	assert.Equal(t, 1, len(noopExecutor.Jobs))
	assert.Equal(t, jobSelected.Id, noopExecutor.Jobs[0].Id)
}

func TestTransportEvents(t *testing.T) {
	ctx := context.Background()
	transport, _, _ := setupTest(t)

	// first let's test submitting a job with an engine we do not have
	spec := &types.JobSpec{
		Engine:   string(executor.EXECUTOR_NOOP),
		Verifier: string(verifier.VERIFIER_NOOP),
		VM: types.JobSpecVm{
			Image:      "image",
			Entrypoint: []string{"entrypoint"},
			Env:        []string{"env"},
		},
		Inputs: []types.StorageSpec{
			{
				Engine: storage.IPFS_DEFAULT,
			},
		},
	}

	deal := &types.JobDeal{
		Concurrency: 1,
	}

	_, err := transport.SubmitJob(ctx, spec, deal)
	assert.NoError(t, err)
	time.Sleep(time.Second * 1)

	expectedEventNames := []string{
		string(types.JOB_EVENT_CREATED),
		string(types.JOB_EVENT_BID),
		string(types.JOB_EVENT_BID_ACCEPTED),
		string(types.JOB_EVENT_RESULTS),
	}
	actualEventNames := []string{}

	for _, event := range transport.Events {
		actualEventNames = append(actualEventNames, string(event.EventName))
	}

	sort.Strings(expectedEventNames)
	sort.Strings(actualEventNames)

	assert.True(t, reflect.DeepEqual(expectedEventNames, actualEventNames), "event list is correct")
}
