package transport_test

import (
	"context"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/computenode"
	"github.com/filecoin-project/bacalhau/pkg/controller"
	"github.com/filecoin-project/bacalhau/pkg/datastore/inmemory"
	"github.com/filecoin-project/bacalhau/pkg/executor"
	executorNoop "github.com/filecoin-project/bacalhau/pkg/executor/noop"
	"github.com/filecoin-project/bacalhau/pkg/localdb/inmemory"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/requesternode"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/transport/inprocess"
	"github.com/filecoin-project/bacalhau/pkg/verifier"
	verifier_noop "github.com/filecoin-project/bacalhau/pkg/verifier/noop"
	"github.com/stretchr/testify/require"

	"github.com/stretchr/testify/suite"
)

type TransportSuite struct {
	suite.Suite
}

// a normal test function and pass our suite to suite.Run
func TestTransportSuite(t *testing.T) {
	suite.Run(t, new(TransportSuite))
}

// Before all suite
func (suite *TransportSuite) SetupAllSuite() {

}

// Before each test
func (suite *TransportSuite) SetupTest() {
	system.InitConfigForTesting(suite.T())
}

func (suite *TransportSuite) TearDownTest() {
}

func (suite *TransportSuite) TearDownAllSuite() {

}

func setupTest(t *testing.T) (
	*inprocess.InProcessTransport,
	*executorNoop.Executor,
	*verifier_noop.Verifier,
	*controller.Controller,
	*system.CleanupManager,
) {
	cm := system.NewCleanupManager()

	noopExecutor, err := executorNoop.NewExecutor()
	require.NoError(t, err)

	noopVerifier, err := verifier_noop.NewVerifier()
	require.NoError(t, err)

	executors := map[executor.EngineType]executor.Executor{
		executor.EngineNoop: noopExecutor,
	}

	verifiers := map[verifier.VerifierType]verifier.Verifier{
		verifier.VerifierNoop: noopVerifier,
	}

	datastore, err := inmemory.NewInMemoryDatastore()
	require.NoError(t, err)

	transport, err := inprocess.NewInprocessTransport()
	require.NoError(t, err)

	ctrl, err := controller.NewController(cm, datastore, transport)
	require.NoError(t, err)

	_, err = computenode.NewComputeNode(
		cm,
		ctrl,
		executors,
		verifiers,
		computenode.NewDefaultComputeNodeConfig(),
	)
	require.NoError(t, err)

	_, err = requesternode.NewRequesterNode(
		cm,
		ctrl,
		verifiers,
		requesternode.RequesterNodeConfig{},
	)
	require.NoError(t, err)

	return transport, noopExecutor, noopVerifier, ctrl, cm
}

func (suite *TransportSuite) TestTransportSanity() {
	cm := system.NewCleanupManager()
	defer cm.Cleanup()
	executors := map[executor.EngineType]executor.Executor{}
	verifiers := map[verifier.VerifierType]verifier.Verifier{}
	datastore, err := inmemory.NewInMemoryDatastore()
	require.NoError(t, err)
	transport, err := inprocess.NewInprocessTransport()
	require.NoError(t, err)
	ctrl, err := controller.NewController(cm, datastore, transport)
	require.NoError(t, err)
	_, err = computenode.NewComputeNode(
		cm,
		ctrl,
		executors,
		verifiers,
		computenode.NewDefaultComputeNodeConfig(),
	)
	require.NoError(t, err)
	_, err = requesternode.NewRequesterNode(
		cm,
		ctrl,
		verifiers,
		requesternode.RequesterNodeConfig{},
	)
	require.NoError(suite.T(), err)
}

func (suite *TransportSuite) TestSchedulerSubmitJob() {
	ctx := context.Background()
	_, noopExecutor, _, ctrl, cm := setupTest(t)
	defer cm.Cleanup()

	spec := executor.JobSpec{
		Engine:   executor.EngineNoop,
		Verifier: verifier.VerifierNoop,
		Docker: executor.JobSpecDocker{
			Image:      "image",
			Entrypoint: []string{"entrypoint"},
			Env:        []string{"env"},
		},
		Inputs: []storage.StorageSpec{
			{
				Engine: storage.StorageSourceIPFS,
			},
		},
	}

	deal := executor.JobDeal{
		Concurrency: 1,
	}

	payload := executor.JobCreatePayload{
		ClientID: "123",
		Spec:     spec,
		Deal:     deal,
	}

	jobSelected, err := ctrl.SubmitJob(ctx, payload)
	require.NoError(t, err)

	time.Sleep(time.Second * 1)
	require.Equal(suite.T(), 1, len(noopExecutor.Jobs))
	require.Equal(suite.T(), jobSelected.ID, noopExecutor.Jobs[0].ID)
}

func (suite *TransportSuite) TestTransportEvents() {
	ctx := context.Background()
	transport, _, _, ctrl, cm := setupTest(t)
	defer cm.Cleanup()

	spec := executor.JobSpec{
		Engine:   executor.EngineNoop,
		Verifier: verifier.VerifierNoop,
		Docker: executor.JobSpecDocker{
			Image:      "image",
			Entrypoint: []string{"entrypoint"},
			Env:        []string{"env"},
		},
		Inputs: []storage.StorageSpec{
			{
				Engine: storage.StorageSourceIPFS,
			},
		},
	}

	deal := executor.JobDeal{
		Concurrency: 1,
	}

	payload := executor.JobCreatePayload{
		ClientID: "123",
		Spec:     spec,
		Deal:     deal,
	}

	_, err := ctrl.SubmitJob(ctx, payload)
	require.NoError(t, err)
	time.Sleep(time.Second * 1)

	expectedEventNames := []string{
		executor.JobEventCreated.String(),
		executor.JobEventBid.String(),
		executor.JobEventBidAccepted.String(),
		executor.JobEventCompleted.String(),
	}
	actualEventNames := []string{}

	for _, event := range transport.GetEvents() {
		actualEventNames = append(actualEventNames, event.EventName.String())
	}

	sort.Strings(expectedEventNames)
	sort.Strings(actualEventNames)

	require.True(suite.T(), reflect.DeepEqual(expectedEventNames, actualEventNames), "event list is correct")
}
