package transport_test

import (
	"context"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/computenode"
	"github.com/filecoin-project/bacalhau/pkg/controller"
	"github.com/filecoin-project/bacalhau/pkg/executor"
	executorNoop "github.com/filecoin-project/bacalhau/pkg/executor/noop"
	"github.com/filecoin-project/bacalhau/pkg/localdb/inmemory"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/publisher"
	publisher_noop "github.com/filecoin-project/bacalhau/pkg/publisher/noop"
	"github.com/filecoin-project/bacalhau/pkg/requesternode"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	storage_noop "github.com/filecoin-project/bacalhau/pkg/storage/noop"
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
	err := system.InitConfigForTesting()
	require.NoError(suite.T(), err)
}

func (suite *TransportSuite) TearDownTest() {
}

func (suite *TransportSuite) TearDownAllSuite() {

}

func setupTest(t *testing.T) (
	*inprocess.InProcessTransport,
	*executorNoop.Executor,
	*verifier_noop.NoopVerifier,
	*controller.Controller,
	*system.CleanupManager,
) {
	cm := system.NewCleanupManager()
	ctx := context.Background()

	noopStorage, err := storage_noop.NewStorageProvider(ctx, cm, storage_noop.StorageConfig{})
	require.NoError(t, err)

	storageProviders := map[model.StorageSourceType]storage.StorageProvider{
		model.StorageSourceIPFS: noopStorage,
	}

	noopExecutor, err := executorNoop.NewExecutor()
	require.NoError(t, err)

	datastore, err := inmemory.NewInMemoryDatastore()
	require.NoError(t, err)

	transport, err := inprocess.NewInprocessTransport()
	require.NoError(t, err)

	ctrl, err := controller.NewController(ctx, cm, datastore, transport, storageProviders)
	require.NoError(t, err)

	noopVerifier, err := verifier_noop.NewNoopVerifier(ctx, cm, ctrl.GetStateResolver())
	require.NoError(t, err)

	noopPublisher, err := publisher_noop.NewNoopPublisher(ctx, cm, ctrl.GetStateResolver())
	require.NoError(t, err)

	executors := map[model.Engine]executor.Executor{
		model.EngineNoop: noopExecutor,
	}

	verifiers := map[model.Verifier]verifier.Verifier{
		model.VerifierNoop: noopVerifier,
	}

	publishers := map[model.Publisher]publisher.Publisher{
		model.PublisherNoop: noopPublisher,
	}

	_, err = computenode.NewComputeNode(
		ctx,
		cm,
		ctrl,
		executors,
		verifiers,
		publishers,
		computenode.NewDefaultComputeNodeConfig(),
	)
	require.NoError(t, err)

	_, err = requesternode.NewRequesterNode(
		ctx,
		cm,
		ctrl,
		verifiers,
		requesternode.RequesterNodeConfig{},
	)
	require.NoError(t, err)

	err = ctrl.Start(ctx)
	require.NoError(t, err)

	err = transport.Start(ctx)
	require.NoError(t, err)

	return transport, noopExecutor, noopVerifier, ctrl, cm
}

func (suite *TransportSuite) TestTransportSanity() {
	cm := system.NewCleanupManager()
	defer cm.Cleanup()
	ctx := context.Background()

	storageProviders := map[model.StorageSourceType]storage.StorageProvider{}
	executors := map[model.Engine]executor.Executor{}
	verifiers := map[model.Verifier]verifier.Verifier{}
	publishers := map[model.Publisher]publisher.Publisher{}
	datastore, err := inmemory.NewInMemoryDatastore()
	require.NoError(suite.T(), err)
	transport, err := inprocess.NewInprocessTransport()
	require.NoError(suite.T(), err)
	ctrl, err := controller.NewController(ctx, cm, datastore, transport, storageProviders)
	require.NoError(suite.T(), err)
	_, err = computenode.NewComputeNode(
		ctx,
		cm,
		ctrl,
		executors,
		verifiers,
		publishers,
		computenode.NewDefaultComputeNodeConfig(),
	)
	require.NoError(suite.T(), err)
	_, err = requesternode.NewRequesterNode(
		ctx,
		cm,
		ctrl,
		verifiers,
		requesternode.RequesterNodeConfig{},
	)
	require.NoError(suite.T(), err)
}

func (suite *TransportSuite) TestSchedulerSubmitJob() {
	ctx := context.Background()
	_, noopExecutor, _, ctrl, cm := setupTest(suite.T())
	defer cm.Cleanup()

	j := &model.Job{}
	j.Spec = model.Spec{
		Engine:   model.EngineNoop,
		Verifier: model.VerifierNoop,
		Docker: model.JobSpecDocker{
			Image:                "image",
			Entrypoint:           []string{"entrypoint"},
			EnvironmentVariables: []string{"env"},
		},
		Inputs: []model.StorageSpec{
			{
				StorageSource: model.StorageSourceIPFS,
			},
		},
	}

	j.Deal = model.Deal{
		Concurrency: 1,
	}

	payload := model.JobCreatePayload{
		ClientID: "123",
		Job:      j,
	}

	jobSelected, err := ctrl.SubmitJob(ctx, payload)
	require.NoError(suite.T(), err)

	time.Sleep(time.Second * 5)
	require.Equal(suite.T(), 1, len(noopExecutor.Jobs))
	require.Equal(suite.T(), jobSelected.ID, noopExecutor.Jobs[0].ID)
}

func (suite *TransportSuite) TestTransportEvents() {
	ctx := context.Background()
	transport, _, _, ctrl, cm := setupTest(suite.T())
	defer cm.Cleanup()

	// Create a new job
	j := &model.Job{}
	j.Spec = model.Spec{
		Engine:    model.EngineNoop,
		Verifier:  model.VerifierNoop,
		Publisher: model.PublisherNoop,
		Docker: model.JobSpecDocker{
			Image:                "image",
			Entrypoint:           []string{"entrypoint"},
			EnvironmentVariables: []string{"env"},
		},
		Inputs: []model.StorageSpec{
			{
				StorageSource: model.StorageSourceIPFS,
			},
		},
	}

	j.Deal = model.Deal{
		Concurrency: 1,
	}

	payload := model.JobCreatePayload{
		ClientID: "123",
		Job:      j,
	}

	_, err := ctrl.SubmitJob(ctx, payload)
	require.NoError(suite.T(), err)
	time.Sleep(time.Second * 1)

	expectedEventNames := []string{
		model.JobEventCreated.String(),
		model.JobEventBid.String(),
		model.JobEventBidAccepted.String(),
		model.JobEventResultsProposed.String(),
		model.JobEventResultsAccepted.String(),
		model.JobEventResultsPublished.String(),
	}
	actualEventNames := []string{}

	time.Sleep(time.Second * 5)

	for _, event := range transport.GetEvents() {
		actualEventNames = append(actualEventNames, event.EventName.String())
	}

	sort.Strings(expectedEventNames)
	sort.Strings(actualEventNames)

	require.True(suite.T(), reflect.DeepEqual(expectedEventNames, actualEventNames), "event list was not equal: %+v != %+v", expectedEventNames, actualEventNames)
}
