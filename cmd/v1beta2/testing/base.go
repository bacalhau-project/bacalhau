package cmdtesting

import (
	"context"
	"time"

	"github.com/stretchr/testify/suite"

	util2 "github.com/bacalhau-project/bacalhau/cmd/v1beta2/util"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/model/v1beta2"
	"github.com/bacalhau-project/bacalhau/pkg/node"
	"github.com/bacalhau-project/bacalhau/pkg/requester/publicapi"
	testutils "github.com/bacalhau-project/bacalhau/pkg/test/utils"
)

type BaseSuite struct {
	suite.Suite
	Node   *node.Node
	Client *publicapi.RequesterAPIClientWrapper
	Host   string
	Port   uint16
}

// before each test
func (s *BaseSuite) SetupTest() {
	logger.ConfigureTestLogging(s.T())
	util2.Fatal = util2.FakeFatalErrorHandler

	ctx := context.Background()
	stack, _ := testutils.SetupTest(ctx, s.T(), 1, 0,
		node.NewComputeConfigWith(node.ComputeConfigParams{
			JobSelectionPolicy: v1beta2.JobSelectionPolicy{
				Locality: v1beta2.Anywhere,
			},
		}),
		node.NewRequesterConfigWith(node.RequesterConfigParams{
			HousekeepingBackgroundTaskInterval: 1 * time.Second,
		}),
	)
	s.Node = stack.Nodes[0]
	s.Host = s.Node.APIServer.Address
	s.Port = s.Node.APIServer.Port
	s.Client = publicapi.NewRequesterAPIClientWrapper(s.Host, s.Port)
}

// After each test
func (s *BaseSuite) TearDownTest() {
	util2.Fatal = util2.FakeFatalErrorHandler
	if s.Node != nil {
		s.Node.CleanupManager.Cleanup(context.Background())
	}
}
