//go:build unit || !integration

package libp2p

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/libp2p"
	"github.com/filecoin-project/bacalhau/pkg/logger"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/multiformats/go-multiaddr"
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

// Before each test
func (suite *Libp2pTransportSuite) SetupTest() {
	logger.ConfigureTestLogging(suite.T())
}

func (suite *Libp2pTransportSuite) TestEncryption() {
	cm := system.NewCleanupManager()
	defer cm.Cleanup()
	ctx := context.Background()

	computeNodePort, err := freeport.GetFreePort()
	require.NoError(suite.T(), err)
	requesterNodePort, err := freeport.GetFreePort()
	require.NoError(suite.T(), err)
	computeNodeHost, err := libp2p.NewHost(computeNodePort)
	require.NoError(suite.T(), err)
	computeNodeTransport, err := NewTransport(ctx, cm, computeNodeHost)
	require.NoError(suite.T(), err)
	computeNodeID := computeNodeTransport.HostID()
	require.NoError(suite.T(), err)
	addr, err := multiaddr.NewMultiaddr(fmt.Sprintf("/ip4/127.0.0.1/tcp/%d/p2p/%s", computeNodePort, computeNodeID))
	require.NoError(suite.T(), err)
	requesterNodeHost, err := libp2p.NewHost(requesterNodePort)
	require.NoError(suite.T(), err)
	requesterNodeTransport, err := NewTransport(ctx, cm, requesterNodeHost)
	require.NoError(suite.T(), err)
	requesterNodeID := requesterNodeTransport.HostID()
	require.NoError(suite.T(), err)

	computeNodeTransport.Subscribe(ctx, func(ctx context.Context, ev model.JobEvent) error {
		return nil
	})
	err = computeNodeTransport.Start(ctx)
	require.NoError(suite.T(), err)

	requesterNodeTransport.Subscribe(ctx, func(ctx context.Context, ev model.JobEvent) error {
		return nil
	})
	err = libp2p.ConnectToPeers(ctx, requesterNodeHost, []multiaddr.Multiaddr{addr})
	require.NoError(suite.T(), err)
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
