//go:build unit || !integration

/*
Copyright 2020 The Kubernetes Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
    http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package version_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/pkg/lib/marshaller"

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

func (suite *VersionSuite) TestVersion() {
	suite.T().Run("client only", func(t *testing.T) {
		t.Run("table output", func(t *testing.T) {
			_, out, err := suite.ExecuteTestCobraCommand("version")
			suite.Require().NoError(err)

			suite.Require().Contains(out, "CLIENT", "Client version not in output")
			suite.Require().NotContains(out, "SERVER", "Server version present in output")
		})
		t.Run("json output", func(t *testing.T) {
			_, out, err := suite.ExecuteTestCobraCommand("version", "--output", string(output.JSONFormat))
			suite.Require().NoError(err, "Could not request version with json output.")

			jsonDoc := &util.Versions{}
			err = marshaller.JSONUnmarshalWithMax([]byte(out), &jsonDoc)
			suite.Require().NoError(err, "Could not unmarshall the output into json - %+v", err)
			suite.Require().NotEmpty(jsonDoc.ClientVersion, "Client version was empty")
			suite.Require().Empty(jsonDoc.ServerVersion, "Server version was not empty")
		})
		t.Run("yaml output", func(t *testing.T) {
			_, out, err := suite.ExecuteTestCobraCommand("version", "--output", string(output.YAMLFormat))
			suite.Require().NoError(err, "Could not request version with json output.")

			yamlDoc := &util.Versions{}
			err = marshaller.YAMLUnmarshalWithMax([]byte(out), &yamlDoc)
			suite.Require().NoError(err, "Could not unmarshall the output into yaml - %+v", err)
			suite.Require().NotEmpty(yamlDoc.ClientVersion.GitCommit, "Client version was empty.")
			suite.Require().Empty(yamlDoc.ServerVersion, "Server version was not empty")
		})
	})
	suite.T().Run("server and client", func(t *testing.T) {
		t.Run("table output", func(t *testing.T) {
			_, out, err := suite.ExecuteTestCobraCommand("version", "--server")
			suite.Require().NoError(err)

			suite.Require().Contains(out, "CLIENT", "Client version not in output")
			suite.Require().Contains(out, "SERVER", "Server version not in output")
		})
		t.Run("json output", func(t *testing.T) {
			_, out, err := suite.ExecuteTestCobraCommand("version", "--server", "--output", string(output.JSONFormat))
			suite.Require().NoError(err, "Could not request version with json output.")

			jsonDoc := &util.Versions{}
			err = marshaller.JSONUnmarshalWithMax([]byte(out), &jsonDoc)
			suite.Require().NoError(err, "Could not unmarshall the output into json - %+v", err)
			suite.Require().Equal(jsonDoc.ClientVersion.GitCommit, jsonDoc.ServerVersion.GitCommit, "Client and Server do not match in json.")
		})
		t.Run("yaml output", func(t *testing.T) {
			_, out, err := suite.ExecuteTestCobraCommand("version", "--server", "--output", string(output.YAMLFormat))
			suite.Require().NoError(err, "Could not request version with json output.")

			yamlDoc := &util.Versions{}
			err = marshaller.YAMLUnmarshalWithMax([]byte(out), &yamlDoc)
			suite.Require().NoError(err, "Could not unmarshall the output into yaml - %+v", err)
			suite.Require().Equal(yamlDoc.ClientVersion.GitCommit, yamlDoc.ServerVersion.GitCommit, "Client and Server do not match in yaml.")
		})
	})
}
