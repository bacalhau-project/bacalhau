//go:build unit || !integration

package transport_test

import (
	"context"
	"reflect"
	"sort"
	"testing"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/devstack"
	"github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/node"
	"github.com/filecoin-project/bacalhau/pkg/requesternode"

	"github.com/filecoin-project/bacalhau/pkg/localdb/inmemory"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/transport/inprocess"
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

// Before each test
func (suite *TransportSuite) SetupTest() {
	logger.ConfigureTestLogging(suite.T())
	err := system.InitConfigForTesting(suite.T())
	require.NoError(suite.T(), err)
}

func setupTest(t *testing.T) *node.Node {
	cm := system.NewCleanupManager()
	ctx := context.Background()

	datastore, err := inmemory.NewInMemoryDatastore()
	require.NoError(t, err)

	transport, err := inprocess.NewInprocessTransport()
	require.NoError(t, err)

	nodeConfig := node.NodeConfig{
		CleanupManager:      cm,
		LocalDB:             datastore,
		Transport:           transport,
		ComputeConfig:       node.NewComputeConfigWithDefaults(),
		RequesterNodeConfig: requesternode.NewDefaultRequesterNodeConfig(),
	}

	node, err := devstack.NewNoopNode(ctx, nodeConfig)
	require.NoError(t, err)

	err = transport.Start(ctx)
	require.NoError(t, err)

	return node
}

func (suite *TransportSuite) TestTransportEvents() {
	ctx := context.Background()
	node := setupTest(suite.T())
	defer node.CleanupManager.Cleanup()

	// Create a new job
	j := &model.Job{}
	j.Spec = model.Spec{
		Engine:    model.EngineNoop,
		Verifier:  model.VerifierNoop,
		Publisher: model.PublisherNoop,
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

	_, err := node.RequesterNode.SubmitJob(ctx, payload)
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

	for _, event := range node.Transport.(*inprocess.InProcessTransport).GetEvents() {
		actualEventNames = append(actualEventNames, event.EventName.String())
	}

	sort.Strings(expectedEventNames)
	sort.Strings(actualEventNames)

	require.True(suite.T(), reflect.DeepEqual(expectedEventNames, actualEventNames), "event list was not equal: %+v != %+v", expectedEventNames, actualEventNames)
}
