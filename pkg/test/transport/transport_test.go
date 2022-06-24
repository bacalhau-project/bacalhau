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

	executors := map[executor.EngineType]executor.Executor{
		executor.EngineNoop: noopExecutor,
	}

	verifiers := map[verifier.VerifierType]verifier.Verifier{
		verifier.VerifierNoop: noopVerifier,
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
	executors := map[executor.EngineType]executor.Executor{}
	verifiers := map[verifier.VerifierType]verifier.Verifier{}
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

	spec := &executor.JobSpec{
		Engine:   executor.EngineNoop,
		Verifier: verifier.VerifierNoop,
		Docker: executor.JobSpecDocker{
			Image:      "image",
			Entrypoint: []string{"entrypoint"},
			Env:        []string{"env"},
		},
		Inputs: []storage.StorageSpec{
			{
				Engine: storage.IPFS_DEFAULT,
			},
		},
	}

	deal := &executor.JobDeal{
		Concurrency: 1,
	}

	jobSelected, err := transport.SubmitJob(ctx, spec, deal)
	assert.NoError(t, err)

	time.Sleep(time.Second * 1)
	assert.Equal(t, 1, len(noopExecutor.Jobs))
	assert.Equal(t, jobSelected.Id, noopExecutor.Jobs[0].Id)
}

func TestTransportEvents(t *testing.T) {
	ctx := context.Background()
	transport, _, _ := setupTest(t)

	spec := &executor.JobSpec{
		Engine:   executor.EngineNoop,
		Verifier: verifier.VerifierNoop,
		Docker: executor.JobSpecDocker{
			Image:      "image",
			Entrypoint: []string{"entrypoint"},
			Env:        []string{"env"},
		},
		Inputs: []storage.StorageSpec{
			{
				Engine: storage.IPFS_DEFAULT,
			},
		},
	}

	deal := &executor.JobDeal{
		Concurrency: 1,
	}

	_, err := transport.SubmitJob(ctx, spec, deal)
	assert.NoError(t, err)
	time.Sleep(time.Second * 1)

	expectedEventNames := []string{
		executor.JobEventCreated.String(),
		executor.JobEventBid.String(),
		executor.JobEventBidAccepted.String(),
		executor.JobEventResults.String(),
	}
	actualEventNames := []string{}

	for _, event := range transport.Events {
		actualEventNames = append(actualEventNames, event.EventName.String())
	}

	sort.Strings(expectedEventNames)
	sort.Strings(actualEventNames)

	assert.True(t, reflect.DeepEqual(expectedEventNames, actualEventNames), "event list is correct")
}
