package bacalhau

import (
	"context"
	"net"
	"net/url"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/node"
	"github.com/filecoin-project/bacalhau/pkg/requester/publicapi"
	testutils "github.com/filecoin-project/bacalhau/pkg/test/utils"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type BaseSuite struct {
	suite.Suite
	node   *node.Node
	client *publicapi.RequesterAPIClient
	host   string
	port   string
}

// before each test
func (s *BaseSuite) SetupTest() {
	logger.ConfigureTestLogging(s.T())
	Fatal = FakeFatalErrorHandler

	ctx := context.Background()
	stack, _ := testutils.SetupTest(ctx, s.T(), 1, 0, false,
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
	s.client = publicapi.NewRequesterAPIClient(s.node.APIServer.GetURI())
	parsedBasedURI, err := url.Parse(s.client.BaseURI)
	require.NoError(s.T(), err)
	host, port, _ := net.SplitHostPort(parsedBasedURI.Host)
	s.host = host
	s.port = port
}

// After each test
func (s *BaseSuite) TearDownTest() {
	Fatal = FatalErrorHandler
	if s.node != nil {
		s.node.CleanupManager.Cleanup(context.Background())
	}
}
