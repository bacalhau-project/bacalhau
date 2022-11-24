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

package bacalhau

import (
	"net"
	"net/url"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/publicapi"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

// In order for 'go test' to run this suite, we need to create
// a normal test function and pass our suite to suite.Run
func TestVersionSuite(t *testing.T) {
	suite.Run(t, new(VersionSuite))
}

type VersionSuite struct {
	suite.Suite
	rootCmd *cobra.Command
}

// Before each test
func (suite *VersionSuite) SetupTest() {
	suite.rootCmd = RootCmd
	logger.ConfigureTestLogging(suite.T())
}

func (suite *VersionSuite) Test_Version() {
	c, cm := publicapi.SetupRequesterNodeForTests(suite.T(), false)
	defer cm.Cleanup()

	parsedBasedURI, _ := url.Parse(c.BaseURI)
	host, port, _ := net.SplitHostPort(parsedBasedURI.Host)
	_, out, err := ExecuteTestCobraCommand(suite.T(), suite.rootCmd, "version",
		"--api-host", host,
		"--api-port", port,
	)
	require.NoError(suite.T(), err)

	require.Contains(suite.T(), string(out), "Client Version", "Client version not in output")
	require.Contains(suite.T(), string(out), "Server Version", "Server version not in output")
}

func (suite *VersionSuite) Test_VersionOutputs() {
	c, cm := publicapi.SetupRequesterNodeForTests(suite.T(), false)
	defer cm.Cleanup()

	parsedBasedURI, _ := url.Parse(c.BaseURI)
	host, port, _ := net.SplitHostPort(parsedBasedURI.Host)
	_, out, err := ExecuteTestCobraCommand(suite.T(), suite.rootCmd, "version",
		"--api-host", host,
		"--api-port", port,
		"--output", JSONFormat,
	)
	require.NoError(suite.T(), err, "Could not request version with json output.")

	jsonDoc := &Versions{}
	err = model.JSONUnmarshalWithMax([]byte(out), &jsonDoc)
	require.NoError(suite.T(), err, "Could not unmarshall the output into json - %+v", err)
	require.Equal(suite.T(), jsonDoc.ClientVersion.GitCommit, jsonDoc.ServerVersion.GitCommit, "Client and Server do not match in json.")

	_, out, err = ExecuteTestCobraCommand(suite.T(), suite.rootCmd, "version",
		"--api-host", host,
		"--api-port", port,
		"--output", YAMLFormat,
	)
	require.NoError(suite.T(), err, "Could not request version with json output.")

	yamlDoc := &Versions{}
	err = model.YAMLUnmarshalWithMax([]byte(out), &yamlDoc)
	require.NoError(suite.T(), err, "Could not unmarshall the output into yaml - %+v", err)
	require.Equal(suite.T(), yamlDoc.ClientVersion.GitCommit, yamlDoc.ServerVersion.GitCommit, "Client and Server do not match in yaml.")

}
