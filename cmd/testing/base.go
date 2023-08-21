package cmdtesting

import (
	"context"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy/semantic"
	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/pkg/devstack"
	noop_executor "github.com/bacalhau-project/bacalhau/pkg/executor/noop"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/node"
	"github.com/bacalhau-project/bacalhau/pkg/requester/publicapi"
	"github.com/bacalhau-project/bacalhau/pkg/test/teststack"
)

type BaseSuite struct {
	suite.Suite
	Node   *node.Node
	Client *publicapi.RequesterAPIClient
	Host   string
	Port   uint16
}

// before each test
func (s *BaseSuite) SetupTest() {
	logger.ConfigureTestLogging(s.T())
	util.Fatal = util.FakeFatalErrorHandler

	ctx := context.Background()
	stack := teststack.Setup(ctx, s.T(),
		devstack.WithNumberOfHybridNodes(1),
		devstack.WithComputeConfig(
			node.NewComputeConfigWith(node.ComputeConfigParams{
				JobSelectionPolicy: node.JobSelectionPolicy{
					Locality: semantic.Anywhere,
				},
			}),
		),
		devstack.WithRequesterConfig(
			node.NewRequesterConfigWith(
				node.RequesterConfigParams{
					HousekeepingBackgroundTaskInterval: 1 * time.Second,
				},
			),
		),
		teststack.WithNoopExecutor(noop_executor.ExecutorConfig{}),
	)
	s.Node = stack.Nodes[0]
	s.Host = s.Node.APIServer.Address
	s.Port = s.Node.APIServer.Port
	s.Client = publicapi.NewRequesterAPIClient(s.Host, s.Port, nil)
}

// After each test
func (s *BaseSuite) TearDownTest() {
	util.Fatal = util.FakeFatalErrorHandler
	if s.Node != nil {
		s.Node.CleanupManager.Cleanup(context.Background())
	}
}
