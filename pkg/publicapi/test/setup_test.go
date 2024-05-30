//go:build unit || !integration

package test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/devstack"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/node"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/client/v2"
	"github.com/bacalhau-project/bacalhau/pkg/setup"
	"github.com/bacalhau-project/bacalhau/pkg/test/teststack"
)

type ServerSuite struct {
	suite.Suite
	server        *publicapi.Server
	client        client.API
	requesterNode *node.Node

	computeNode   *node.Node
	computeClient client.API
}

func (s *ServerSuite) SetupSuite() {
	logger.ConfigureTestLogging(s.T())
	fsr, c := setup.SetupBacalhauRepoForTesting(s.T())

	ctx := context.Background()

	stack := teststack.Setup(ctx, s.T(), fsr, c,
		devstack.WithNumberOfRequesterOnlyNodes(1),
		devstack.WithNumberOfComputeOnlyNodes(1),
		devstack.WithDependencyInjector(devstack.NewNoopNodeDependencyInjector()),
		devstack.WithAutoNodeApproval(),
	)

	s.requesterNode = stack.Nodes[0]
	s.client = client.New(s.requesterNode.APIServer.GetURI().String())

	s.computeNode = stack.Nodes[1]
	s.computeClient = client.New(s.computeNode.APIServer.GetURI().String())

	s.Require().NoError(WaitForAlive(ctx, s.client))
	s.Require().NoError(WaitForAlive(ctx, s.computeClient))
}

func (s *ServerSuite) TearDownSuite() {
	if s.server != nil {
		s.Require().NoError(s.server.Shutdown(context.Background()))
	}
}

func TestServerSuite(t *testing.T) {
	suite.Run(t, new(ServerSuite))
}
