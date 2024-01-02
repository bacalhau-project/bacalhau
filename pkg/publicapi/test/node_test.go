//go:build unit || !integration

package test

import (
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"
)

func (s *ServerSuite) TestNodeList() {
	resp, err := s.client.Nodes().List(&apimodels.ListNodesRequest{})
	require.NoError(s.T(), err)
	require.NotNil(s.T(), resp)
	require.NotEmpty(s.T(), resp.Nodes)
	require.Equal(s.T(), 2, len(resp.Nodes))
}

func (s *ServerSuite) TestNodeListLabels() {
	req1, err := labels.NewRequirement("name", selection.Equals, []string{"node-1"})
	require.NoError(s.T(), err)
	req2, err := labels.NewRequirement("env", selection.Equals, []string{"devstack"})
	require.NoError(s.T(), err)

	resp, err := s.client.Nodes().List(&apimodels.ListNodesRequest{
		Labels: []labels.Requirement{*req1, *req2},
	})
	require.NoError(s.T(), err)
	require.NotNil(s.T(), resp)
	require.NotEmpty(s.T(), resp.Nodes)
	require.Equal(s.T(), 1, len(resp.Nodes))
}
