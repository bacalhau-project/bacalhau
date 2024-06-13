//go:build unit || !integration

package agent_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/lib/marshaller"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"

	cmdtesting "github.com/bacalhau-project/bacalhau/cmd/testing"
	"github.com/bacalhau-project/bacalhau/cmd/util/output"
)

func TestAliveSuite(t *testing.T) {
	suite.Run(t, new(AliveSuite))
}

type AliveSuite struct {
	cmdtesting.BaseSuite
}

func (s *AliveSuite) TestAliveJSONOutput() {
	_, out, err := s.ExecuteTestCobraCommand("agent", "alive",
		"--output", string(output.JSONFormat),
	)
	s.Require().NoError(err, "Could not request alive with json output.")

	aliveInfo := &apimodels.IsAliveResponse{}
	err = marshaller.JSONUnmarshalWithMax([]byte(out), &aliveInfo)
	s.Require().NoError(err, "Could not unmarshall the output into json - %+v", err)
	s.Require().True(aliveInfo.IsReady())
}

func (s *AliveSuite) TestAliveYAMLOutput() {
	_, out, err := s.ExecuteTestCobraCommand("agent", "alive")
	s.Require().NoError(err, "Could not request alive with yaml output.")

	aliveInfo := &apimodels.IsAliveResponse{}
	err = marshaller.YAMLUnmarshalWithMax([]byte(out), &aliveInfo)
	s.Require().NoError(err, "Could not unmarshall the output into yaml - %+v", out)
	s.Require().True(aliveInfo.IsReady())
}
