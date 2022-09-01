package libp2p

import (
	"context"
	"fmt"
	"testing"
	"time"

	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/phayes/freeport"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type Libp2pTransportSuite struct {
	suite.Suite
}

// a normal test function and pass our suite to suite.Run
func TestLibp2pTransportSuite(t *testing.T) {
	suite.Run(t, new(Libp2pTransportSuite))
}

// Before all suite
func (suite *Libp2pTransportSuite) SetupAllSuite() {

}

// Before each test
func (suite *Libp2pTransportSuite) SetupTest() {

}

func (suite *Libp2pTransportSuite) TearDownTest() {
}

func (suite *Libp2pTransportSuite) TearDownAllSuite() {

}

func (suite *Libp2pTransportSuite) TestEncryption() {
	TestData := "hello encryption my old friend"
	cm := system.NewCleanupManager()
	defer cm.Cleanup()
	ctx := context.Background()

	computeNodePort, err := freeport.GetFreePort()
	require.NoError(suite.T(), err)
	requesterNodePort, err := freeport.GetFreePort()
	require.NoError(suite.T(), err)
	computeNodeTransport, err := NewTransport(ctx, cm, computeNodePort, []string{})
	require.NoError(suite.T(), err)
	computeNodeID, err := computeNodeTransport.HostID(ctx)
	require.NoError(suite.T(), err)
	requesterNodeTransport, err := NewTransport(ctx, cm, requesterNodePort, []string{
		fmt.Sprintf("/ip4/127.0.0.1/tcp/%d/p2p/%s", computeNodePort, computeNodeID),
	})
	require.NoError(suite.T(), err)
	requesterNodeID, err := requesterNodeTransport.HostID(ctx)
	require.NoError(suite.T(), err)

	computeNodeTransport.Subscribe(func(ctx context.Context, ev model.JobEvent) {
		if ev.EventName == model.JobEventBidAccepted {
			encryptedData, err := computeNodeTransport.Encrypt(ctx, []byte(TestData), ev.SenderPublicKey)
			require.NoError(suite.T(), err)
			err = computeNodeTransport.Publish(ctx, model.JobEvent{
				EventName:            model.JobEventResultsProposed,
				SourceNodeID:         computeNodeID,
				TargetNodeID:         requesterNodeID,
				VerificationProposal: encryptedData,
			})
			require.NoError(suite.T(), err)
		}
	})
	err = computeNodeTransport.Start(ctx)
	require.NoError(suite.T(), err)

	requesterNodeTransport.Subscribe(func(ctx context.Context, ev model.JobEvent) {
		if ev.EventName == model.JobEventResultsProposed {
			decryptedData, err := requesterNodeTransport.Decrypt(ctx, ev.VerificationProposal)
			require.NoError(suite.T(), err)
			require.Equal(suite.T(), TestData, string(decryptedData), "the decrypted data should be the same as the original data")
		}
	})
	err = requesterNodeTransport.Start(ctx)
	require.NoError(suite.T(), err)

	time.Sleep(time.Second * 1)

	err = requesterNodeTransport.Publish(ctx, model.JobEvent{
		EventName:    model.JobEventBidAccepted,
		SourceNodeID: requesterNodeID,
		TargetNodeID: computeNodeID,
	})
	require.NoError(suite.T(), err)
}
