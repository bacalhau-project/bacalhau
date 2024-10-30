//go:build unit || !integration

package agent_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	cmdtesting "github.com/bacalhau-project/bacalhau/cmd/testing"
	"github.com/bacalhau-project/bacalhau/cmd/util/output"
	"github.com/bacalhau-project/bacalhau/pkg/lib/marshaller"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
)

func TestConfigSuite(t *testing.T) {
	suite.Run(t, new(ConfigSuite))
}

type ConfigSuite struct {
	cmdtesting.BaseSuite
}

func (s *ConfigSuite) TestConfigJSONOutput() {
	_, out, err := s.ExecuteTestCobraCommand("agent", "config",
		"--output", string(output.JSONFormat),
	)
	s.Require().NoError(err, "Could not request config with json output.")

	resp := &apimodels.GetAgentConfigResponse{}
	err = marshaller.JSONUnmarshalWithMax([]byte(out), &resp)
	s.Require().NoError(err, "Could not unmarshal the output into json - %+v", err)
	s.Require().True(resp.Config.Orchestrator.Enabled)
}

func (s *ConfigSuite) TestConfigYAMLOutput() {
	_, out, err := s.ExecuteTestCobraCommand("agent", "config")
	s.Require().NoError(err, "Could not request config with yaml output.")

	resp := &apimodels.GetAgentConfigResponse{}
	err = marshaller.YAMLUnmarshalWithMax([]byte(out), &resp)
	s.Require().NoError(err, "Could not unmarshal the output into yaml - %+v", out)
	s.Require().True(resp.Config.Orchestrator.Enabled)
}
