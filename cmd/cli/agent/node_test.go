//go:build unit || !integration

package agent_test

import (
	"fmt"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/lib/marshaller"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/stretchr/testify/suite"

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
	_, out, err := cmdtesting.ExecuteTestCobraCommand("agent", "node",
		"--api-host", s.Host,
		"--api-port", fmt.Sprint(s.Port),
		"--output", string(output.JSONFormat),
	)
	s.Require().NoError(err, "Could not request node with json output.")

	nodeInfo := &models.NodeInfo{}
	err = marshaller.JSONUnmarshalWithMax([]byte(out), &nodeInfo)
	s.Require().NoError(err, "Could not unmarshall the output into json - %+v", err)
	s.Require().Equal(s.Node.ID, nodeInfo.ID(), "Node ID does not match in json.")
}

func (s *NodeSuite) TestNodeYAMLOutput() {
	_, out, err := cmdtesting.ExecuteTestCobraCommand("agent", "node",
		"--api-host", s.Host,
		"--api-port", fmt.Sprint(s.Port),
	)
	s.Require().NoError(err, "Could not request node with yaml output.")

	nodeInfo := &models.NodeInfo{}
	err = marshaller.YAMLUnmarshalWithMax([]byte(out), &nodeInfo)
	s.Require().NoError(err, "Could not unmarshall the output into yaml - %+v", err)
	s.Require().Equal(s.Node.ID, nodeInfo.ID(), "Node ID does not match in yaml.")
}
