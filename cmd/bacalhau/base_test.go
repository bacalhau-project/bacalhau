package bacalhau

import (
	"context"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/node"
	"github.com/bacalhau-project/bacalhau/pkg/requester/publicapi"
	testutils "github.com/bacalhau-project/bacalhau/pkg/test/utils"
)

type BaseSuite struct {
	suite.Suite
	node   *node.Node
	client *publicapi.RequesterAPIClient
	host   string
	port   uint16
}

// before each test
func (s *BaseSuite) SetupTest() {
	logger.ConfigureTestLogging(s.T())
	Fatal = FakeFatalErrorHandler

	ctx := context.Background()
	stack, _ := testutils.SetupTest(ctx, s.T(), 1, 0,
		node.NewComputeConfigWith(node.ComputeConfigParams{
			JobSelectionPolicy: model.JobSelectionPolicy{
				Locality: model.Anywhere,
			},
		}),
		node.NewRequesterConfigWith(node.RequesterConfigParams{
			HousekeepingBackgroundTaskInterval: 1 * time.Second,
		}),
	)
	s.node = stack.Nodes[0]
	s.host = s.node.APIServer.Address
	s.port = s.node.APIServer.Port
	s.client = publicapi.NewRequesterAPIClient(s.host, s.port)
}

// After each test
func (s *BaseSuite) TearDownTest() {
	Fatal = FatalErrorHandler
	if s.node != nil {
		s.node.CleanupManager.Cleanup(context.Background())
	}
}
