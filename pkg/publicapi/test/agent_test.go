//go:build unit || !integration

package test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bacalhau-project/bacalhau/pkg/models"
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
	s.Require().NotNil(resp.NodeInfo)

	node := s.requesterNode
	expectedNode, err := node.RequesterNode.NodeInfoStore.Get(context.Background(), s.requesterNode.ID)
	s.Require().NoError(err)
	equalNodeInfo(s.T(), expectedNode, *resp.NodeInfo)
}

func (s *ServerSuite) TestAgentNodeCompute() {
	ctx := context.Background()
	resp, err := s.computeClient.Agent().Node(ctx, &apimodels.GetAgentNodeRequest{})
	s.Require().NoError(err)
	s.Require().NotEmpty(resp)
	s.Require().NotNil(resp.NodeInfo)
}

func equalNodeInfo(t *testing.T, a, b models.NodeInfo) {
	require.Equal(t, a.BacalhauVersion, b.BacalhauVersion)
	require.Equal(t, a.ID(), b.ID())
	require.Equal(t, a.NodeType, b.NodeType)
	require.Equal(t, a.Labels, b.Labels)

	if a.ComputeNodeInfo == nil {
		require.Nil(t, b.ComputeNodeInfo)
		return
	}
	require.ElementsMatch(t, a.ComputeNodeInfo.ExecutionEngines, b.ComputeNodeInfo.ExecutionEngines)
	require.ElementsMatch(t, a.ComputeNodeInfo.Publishers, b.ComputeNodeInfo.Publishers)
	require.ElementsMatch(t, a.ComputeNodeInfo.StorageSources, b.ComputeNodeInfo.StorageSources)
	require.Equal(t, a.ComputeNodeInfo.MaxCapacity, b.ComputeNodeInfo.MaxCapacity)
	require.Equal(t, a.ComputeNodeInfo.AvailableCapacity, b.ComputeNodeInfo.AvailableCapacity)
	require.Equal(t, a.ComputeNodeInfo.MaxJobRequirements, b.ComputeNodeInfo.MaxJobRequirements)
	require.Equal(t, a.ComputeNodeInfo.RunningExecutions, b.ComputeNodeInfo.RunningExecutions)
	require.Equal(t, a.ComputeNodeInfo.RunningExecutions, b.ComputeNodeInfo.RunningExecutions)

}
