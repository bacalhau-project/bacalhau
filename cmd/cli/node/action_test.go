//go:build unit || !integration

package node_test

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	cmdtesting "github.com/bacalhau-project/bacalhau/cmd/testing"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/setup"
)

type NodeActionSuite struct {
	cmdtesting.BaseSuite
}

func TestNodeActionSuite(t *testing.T) {
	suite.Run(t, new(NodeActionSuite))
}

func (s *NodeActionSuite) SetupSuite() {
	logger.ConfigureTestLogging(s.T())
	setup.SetupBacalhauRepoForTesting(s.T())
}

func (s *NodeActionSuite) TestListNodes() {
	// Get default states for the test node
	_, out, err := s.ExecuteTestCobraCommand(
		"node",
		"list",
		"--output", "csv",
	)
	s.Require().NoError(err)

	cells := getCells(out, 1)
	s.Require().Equal("APPROVED", cells[2], "Expected the node to be approved")

	nodeID := cells[0]

	// Try to approve, expect failure
	_, out, err = s.ExecuteTestCobraCommand(
		"node",
		"approve",
		nodeID,
	)
	s.Require().NoError(err)
	s.Require().Contains(out, "node already approved")
	s.Require().Contains(out, nodeID)

	// Now reject the node
	_, out, err = s.ExecuteTestCobraCommand(
		"node",
		"reject",
		nodeID,
	)
	s.Require().NoError(err)
	s.Require().Contains(out, "Ok")

	// Try to reject again - expect failure
	_, out, err = s.ExecuteTestCobraCommand(
		"node",
		"reject",
		nodeID,
	)
	s.Require().NoError(err)
	s.Require().Contains(out, "node already rejected")

	// Set it to approve again
	_, out, err = s.ExecuteTestCobraCommand(
		"node",
		"approve",
		nodeID,
	)
	s.Require().NoError(err)
	s.Require().Contains(out, "Ok")

	// Delete the node
	_, out, err = s.ExecuteTestCobraCommand(
		"node",
		"delete",
		nodeID,
	)
	s.Require().NoError(err)
	s.Require().Contains(out, "Ok")

	_, out, err = s.ExecuteTestCobraCommand(
		"node",
		"list",
		"--output", "csv",
	)
	s.Require().NoError(err)
	s.Require().NotContains(out, nodeID)
}

func getCells(output string, lineNo int) []string {
	lines := strings.Split(output, "\n")
	line := lines[lineNo]
	return strings.Split(line, ",")
}
