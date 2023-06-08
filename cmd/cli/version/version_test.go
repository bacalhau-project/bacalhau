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
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/cmd/cli/version"
	cmdtesting "github.com/bacalhau-project/bacalhau/cmd/testing"
	"github.com/bacalhau-project/bacalhau/cmd/util/handler"
	"github.com/bacalhau-project/bacalhau/pkg/model"
)

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestVersionSuite(t *testing.T) {
	suite.Run(t, new(VersionSuite))
}

type VersionSuite struct {
	cmdtesting.BaseSuite
}

func (suite *VersionSuite) Test_Version() {
	_, out, err := cmdtesting.ExecuteTestCobraCommand("version",
		"--api-host", suite.Host,
		"--api-port", fmt.Sprint(suite.Port),
	)
	require.NoError(suite.T(), err)

	require.Contains(suite.T(), out, "Client Version", "Client version not in output")
	require.Contains(suite.T(), out, "Server Version", "Server version not in output")
}

func (suite *VersionSuite) Test_VersionOutputs() {
	_, out, err := cmdtesting.ExecuteTestCobraCommand("version",
		"--api-host", suite.Host,
		"--api-port", fmt.Sprint(suite.Port),
		"--output", handler.JSONFormat,
	)
	require.NoError(suite.T(), err, "Could not request version with json output.")

	jsonDoc := &version.Versions{}
	err = model.JSONUnmarshalWithMax([]byte(out), &jsonDoc)
	require.NoError(suite.T(), err, "Could not unmarshall the output into json - %+v", err)
	require.Equal(suite.T(), jsonDoc.ClientVersion.GitCommit, jsonDoc.ServerVersion.GitCommit, "Client and Server do not match in json.")

	_, out, err = cmdtesting.ExecuteTestCobraCommand("version",
		"--api-host", suite.Host,
		"--api-port", fmt.Sprint(suite.Port),
		"--output", handler.YAMLFormat,
	)
	require.NoError(suite.T(), err, "Could not request version with json output.")

	yamlDoc := &version.Versions{}
	err = model.YAMLUnmarshalWithMax([]byte(out), &yamlDoc)
	require.NoError(suite.T(), err, "Could not unmarshall the output into yaml - %+v", err)
	require.Equal(suite.T(), yamlDoc.ClientVersion.GitCommit, yamlDoc.ServerVersion.GitCommit, "Client and Server do not match in yaml.")

}
