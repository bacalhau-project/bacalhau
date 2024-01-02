//go:build unit || !integration

package agent_test

import (
	"fmt"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/lib/marshaller"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/version"
	"github.com/stretchr/testify/suite"

	cmdtesting "github.com/bacalhau-project/bacalhau/cmd/testing"
	"github.com/bacalhau-project/bacalhau/cmd/util/output"
)

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestVersionSuite(t *testing.T) {
	suite.Run(t, new(VersionSuite))
}

type VersionSuite struct {
	cmdtesting.BaseSuite
}

func (s *VersionSuite) TestVersionHumanOutput() {
	_, out, err := cmdtesting.ExecuteTestCobraCommand("agent", "version",
		"--api-host", s.Host,
		"--api-port", fmt.Sprint(s.Port),
	)
	s.Require().NoError(err)

	expectedVersion := version.Get()
	s.Require().Contains(out, expectedVersion.GitVersion, "GitVersion info not in output")
	s.Require().Contains(out, expectedVersion.BuildDate.String(), "BuildDate info not in output")
	s.Require().Contains(out, expectedVersion.GitCommit, "GitCommit info not in output")
	s.Require().Contains(out, "Bacalhau", "Bacalhau name not in output")
}

func (s *VersionSuite) TestVersionJSONOutput() {
	_, out, err := cmdtesting.ExecuteTestCobraCommand("agent", "version",
		"--api-host", s.Host,
		"--api-port", fmt.Sprint(s.Port),
		"--output", string(output.JSONFormat),
	)
	s.Require().NoError(err, "Could not request version with json output.")

	expectedVersion := version.Get()
	printedVersion := &models.BuildVersionInfo{}
	err = marshaller.JSONUnmarshalWithMax([]byte(out), &printedVersion)
	s.Require().NoError(err, "Could not unmarshall the output into json - %+v", err)
	s.Require().Equal(expectedVersion, printedVersion, "Versions do not match in json.")
}

func (s *VersionSuite) TestVersionYAMLOutput() {
	_, out, err := cmdtesting.ExecuteTestCobraCommand("agent", "version",
		"--api-host", s.Host,
		"--api-port", fmt.Sprint(s.Port),
		"--output", string(output.YAMLFormat),
	)
	s.Require().NoError(err, "Could not request version with json output.")

	expectedVersion := version.Get()
	printedVersion := &models.BuildVersionInfo{}
	err = marshaller.YAMLUnmarshalWithMax([]byte(out), &printedVersion)
	s.Require().NoError(err, "Could not unmarshall the output into yaml - %+v", err)
	s.Require().Equal(expectedVersion, printedVersion, "Versions do not match in yaml.")
}
