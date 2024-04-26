//go:build unit || !integration

package test

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	"github.com/bacalhau-project/bacalhau/pkg/version"
)

func (s *ServerSuite) TestAlive() {
	ctx := context.Background()
	resp, err := s.client.Agent().Alive(ctx)
	s.Require().NoError(err)
	s.Require().Equal("OK", resp.Status)
	s.Require().True(resp.IsReady())
}

func (s *ServerSuite) TestAgentVersion() {
	ctx := context.Background()
	resp, err := s.client.Agent().Version(ctx)
	s.Require().NoError(err)
	s.Require().NotEmpty(resp)
	s.Require().NotNil(resp.BuildVersionInfo)
	s.Require().Equal(version.Get(), resp.BuildVersionInfo)

}

func (s *ServerSuite) TestAgentNode() {
	ctx := context.Background()
	resp, err := s.client.Agent().Node(ctx, &apimodels.GetAgentNodeRequest{})
	s.Require().NoError(err)
	s.Require().NotEmpty(resp)
	s.Require().NotNil(resp.NodeState)

	requesterNode := s.requesterNode
	// NB(forrest): we are only asserting NodeInfos are equal (which excludes approvals and liveness from NodeState)
	// because we are asking the requester's NodeInfoStore for the NodeState it contains on itself (s.requesterNode.ID)
	// and since the requester doesn't send heartbeat messages to itself it will consider itself disconnected
	expectedNode, err := requesterNode.RequesterNode.NodeInfoStore.Get(context.Background(), s.requesterNode.ID)
	s.Require().NoError(err)
	s.Require().Equal(expectedNode.Info, resp.Info)
}

func (s *ServerSuite) TestAgentNodeCompute() {
	ctx := context.Background()
	resp, err := s.computeClient.Agent().Node(ctx, &apimodels.GetAgentNodeRequest{})
	s.Require().NoError(err)
	s.Require().NotEmpty(resp)
	s.Require().NotNil(resp.NodeState)
}
