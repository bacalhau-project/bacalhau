package bacalhau

import (
	"context"
	"net"
	"net/url"
	"strings"
	"testing"

	"github.com/filecoin-project/bacalhau/pkg/publicapi"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type CreateSuite struct {
	suite.Suite
	rootCmd *cobra.Command
}

func TestCreateSuite(t *testing.T) {
	suite.Run(t, new(CreateSuite))
}

//before all the suite
func (suite *CreateSuite) SetupSuite() {

}

//before each test
func (suite *CreateSuite) SetupTest() {
	require.NoError(suite.T(), system.InitConfigForTesting())
	suite.rootCmd = RootCmd
}

func (suite *CreateSuite) TearDownTest() {

}

func (suite *CreateSuite) TearDownAllSuite() {

}

func (suite *CreateSuite) TestCreateJSON_GenericSubmit() {
	tests := []struct {
		numberOfJobs int
	}{
		{numberOfJobs: 1}, // Test for one
		{numberOfJobs: 5}, // Test for five
	}

	for i, tc := range tests {
		func() {
			ctx := context.Background()
			c, cm := publicapi.SetupTests(suite.T())
			defer cm.Cleanup()

			*OC = *NewCreateOptions()

			parsedBasedURI, err := url.Parse(c.BaseURI)
			require.NoError(suite.T(), err)

			host, port, _ := net.SplitHostPort(parsedBasedURI.Host)
			_, out, err := ExecuteTestCobraCommand(suite.T(), suite.rootCmd, "create",
				"--api-host", host,
				"--api-port", port,
				"../../testdata/job.json",
			)
			require.NoError(suite.T(), err, "Error submitting job. Run - Number of Jobs: %d. Job number: %d", tc.numberOfJobs, i)

			job, _, err := c.Get(ctx, strings.TrimSpace(out))
			require.NoError(suite.T(), err)
			require.NotNil(suite.T(), job, "Failed to get job with ID: %s", out)
		}()
	}
}

func (suite *CreateSuite) TestCreateYAML_GenericSubmit() {
	tests := []struct {
		numberOfJobs int
	}{
		{numberOfJobs: 1}, // Test for one
		{numberOfJobs: 5}, // Test for five
	}

	for i, tc := range tests {

		testFiles := []string{"../../testdata/job.yaml", "../../testdata/job-url.yaml"}

		for _, testFile := range testFiles {
			func() {
				ctx := context.Background()
				c, cm := publicapi.SetupTests(suite.T())
				defer cm.Cleanup()

				*OC = *NewCreateOptions()

				parsedBasedURI, err := url.Parse(c.BaseURI)
				require.NoError(suite.T(), err)

				host, port, _ := net.SplitHostPort(parsedBasedURI.Host)
				_, out, err := ExecuteTestCobraCommand(suite.T(), suite.rootCmd, "create",
					"--api-host", host,
					"--api-port", port,
					testFile,
				)

				require.NoError(suite.T(), err, "Error submitting job. Run - Number of Jobs: %d. Job number: %d", tc.numberOfJobs, i)

				job, _, err := c.Get(ctx, strings.TrimSpace(out))
				require.NoError(suite.T(), err)
				require.NotNil(suite.T(), job, "Failed to get job with ID: %s", out)
			}()
		}
	}
}
