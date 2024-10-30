//go:build unit || !integration

package agent_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/suite"
	"gopkg.in/yaml.v3"

	cmdtesting "github.com/bacalhau-project/bacalhau/cmd/testing"
	"github.com/bacalhau-project/bacalhau/cmd/util/output"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
)

func TestConfigSuite(t *testing.T) {
	suite.Run(t, new(ConfigSuite))
}

type ConfigSuite struct {
	cmdtesting.BaseSuite
}

func (s *ConfigSuite) TestConfigJSONOutput() {
	_, out, err := s.ExecuteTestCobraCommand(
		"agent", "config", "--output", string(output.JSONFormat), "--pretty=false",
	)
	s.Require().NoError(err, "Could not request config with json output.")

	var cfg types.Bacalhau
	err = json.Unmarshal([]byte(out), &cfg)
	s.Require().NoError(err, "Could not unmarshal the output into json - %+v", err)
	s.Require().True(cfg.Orchestrator.Enabled)
}

func (s *ConfigSuite) TestConfigYAMLOutput() {
	// NB: the default output is yaml, thus we don't specify it here.
	_, out, err := s.ExecuteTestCobraCommand("agent", "config")
	s.Require().NoError(err, "Could not request config with yaml output.")

	var cfg types.Bacalhau
	err = yaml.Unmarshal([]byte(out), &cfg)
	s.Require().NoError(err, "Could not unmarshal the output into yaml - %+v", out)
	s.Require().True(cfg.Orchestrator.Enabled)
}
