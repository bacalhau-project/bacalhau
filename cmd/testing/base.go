package cmdtesting

import (
	"context"
	"fmt"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy/semantic"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/client"
	clientv2 "github.com/bacalhau-project/bacalhau/pkg/publicapi/client/v2"

	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/pkg/devstack"
	noop_executor "github.com/bacalhau-project/bacalhau/pkg/executor/noop"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/node"
	"github.com/bacalhau-project/bacalhau/pkg/test/teststack"
)

type BaseSuite struct {
	suite.Suite
	Node     *node.Node
	Client   *client.APIClient
	ClientV2 *clientv2.Client
	Host     string
	Port     uint16
}

// before each test
func (s *BaseSuite) SetupTest() {
	logger.ConfigureTestLogging(s.T())
	util.Fatal = util.FakeFatalErrorHandler

	computeConfig, err := node.NewComputeConfigWith(node.ComputeConfigParams{
		JobSelectionPolicy: node.JobSelectionPolicy{
			Locality: semantic.Anywhere,
		},
	})
	s.Require().NoError(err)
	ctx := context.Background()
	requesterConfig, err := node.NewRequesterConfigWith(
		node.RequesterConfigParams{
			HousekeepingBackgroundTaskInterval: 1 * time.Second,
		},
	)
	s.Require().NoError(err)
	stack := teststack.Setup(ctx, s.T(),
		devstack.WithNumberOfHybridNodes(1),
		devstack.WithComputeConfig(computeConfig),
		devstack.WithRequesterConfig(requesterConfig),
		teststack.WithNoopExecutor(noop_executor.ExecutorConfig{}),
	)
	s.Node = stack.Nodes[0]
	s.Host = s.Node.APIServer.Address
	s.Port = s.Node.APIServer.Port
	s.Client = client.NewAPIClient(client.NoTLS, s.Host, s.Port)
	s.ClientV2 = clientv2.New(clientv2.Options{
		Address: fmt.Sprintf("http://%s:%d", s.Host, s.Port),
	})
}

// After each test
func (s *BaseSuite) TearDownTest() {
	util.Fatal = util.FakeFatalErrorHandler
	if s.Node != nil {
		s.Node.CleanupManager.Cleanup(context.Background())
	}
}
