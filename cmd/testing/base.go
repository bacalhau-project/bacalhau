package cmdtesting

import (
	"context"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/pkg/devstack"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/node"
	"github.com/bacalhau-project/bacalhau/pkg/requester/publicapi"
	testutils "github.com/bacalhau-project/bacalhau/pkg/test/utils"
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
	stack := testutils.SetupTestDevStack(ctx, s.T(),
		devstack.WithNumberOfHybridNodes(1),
		devstack.WithComputeConfig(
			node.NewComputeConfigWith(node.ComputeConfigParams{
				JobSelectionPolicy: model.JobSelectionPolicy{
					Locality: model.Anywhere,
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
		devstack.WithDependencyInjector(node.NodeDependencyInjector{
			ExecutorsFactory: &testutils.MixedExecutorFactory{
				StandardFactory: node.NewStandardExecutorsFactory(),
				NoopFactory:     devstack.NewNoopExecutorsFactory(),
			},
		}),
	)
	s.Node = stack.Nodes[0]
	s.Host = s.Node.APIServer.Address
	s.Port = s.Node.APIServer.Port
	s.Client = publicapi.NewRequesterAPIClient(s.Host, s.Port)
}

// After each test
func (s *BaseSuite) TearDownTest() {
	util.Fatal = util.FakeFatalErrorHandler
	if s.Node != nil {
		s.Node.CleanupManager.Cleanup(context.Background())
	}
}
