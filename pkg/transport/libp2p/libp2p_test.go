package libp2p

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/executor"
	_ "github.com/filecoin-project/bacalhau/pkg/logger"
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

func (suite *Libp2pTransportSuite) TestTransportSanity() {
	cm := system.NewCleanupManager()
	defer cm.Cleanup()
	ctx := context.Background()

	portA, err := freeport.GetFreePort()
	require.NoError(suite.T(), err)
	portB, err := freeport.GetFreePort()
	require.NoError(suite.T(), err)
	transportA, err := NewTransport(cm, portA, []string{})
	require.NoError(suite.T(), err)
	idA, err := transportA.HostID(ctx)
	require.NoError(suite.T(), err)
	transportB, err := NewTransport(cm, portB, []string{
		fmt.Sprintf("/ip4/127.0.0.1/tcp/%d/p2p/%s", portA, idA),
	})
	require.NoError(suite.T(), err)
	idB, err := transportB.HostID(ctx)
	require.NoError(suite.T(), err)

	transportA.Subscribe(func(ctx context.Context, ev executor.JobEvent) {
		fmt.Printf("ev A --------------------------------------\n")
		//spew.Dump(ev)
	})
	err = transportA.Start(ctx)
	require.NoError(suite.T(), err)

	transportB.Subscribe(func(ctx context.Context, ev executor.JobEvent) {
		fmt.Printf("ev B --------------------------------------\n")
		//spew.Dump(ev)
	})
	err = transportB.Start(ctx)
	require.NoError(suite.T(), err)

	time.Sleep(time.Second * 1)

	err = transportA.Publish(ctx, executor.JobEvent{
		SourceNodeID: idA,
		TargetNodeID: idB,
	})
	require.NoError(suite.T(), err)
}
