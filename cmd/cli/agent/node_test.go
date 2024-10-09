//go:build unit || !integration

package agent_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/lib/marshaller"
	"github.com/bacalhau-project/bacalhau/pkg/models"

	cmdtesting "github.com/bacalhau-project/bacalhau/cmd/testing"
	"github.com/bacalhau-project/bacalhau/cmd/util/output"
)

func TestNodeSuite(t *testing.T) {
	suite.Run(t, new(NodeSuite))
}

type NodeSuite struct {
	cmdtesting.BaseSuite
}

func (s *NodeSuite) TestNodeJSONOutput() {
	_, out, err := s.ExecuteTestCobraCommand("agent", "node", "--output", string(output.JSONFormat))
	s.Require().NoError(err, "Could not request node with json output.")

	nodeInfo := &models.NodeState{}
	err = marshaller.JSONUnmarshalWithMax([]byte(out), &nodeInfo)
	s.Require().NoError(err, "Could not unmarshal the output into json - %+v", out)
	s.Require().Equal(s.Node.ID, nodeInfo.Info.ID(), "Node ID does not match in json.")
}

func (s *NodeSuite) TestNodeYAMLOutput() {
	_, out, err := s.ExecuteTestCobraCommand("agent", "node")
	s.Require().NoError(err, "Could not request node with yaml output.")

	nodeInfo := &models.NodeState{}
	err = marshaller.YAMLUnmarshalWithMax([]byte(out), &nodeInfo)
	s.Require().NoError(err, "Could not unmarshal the output into yaml - %+v", out)
	s.Require().Equal(s.Node.ID, nodeInfo.Info.ID(), "Node ID does not match in yaml.")
}
