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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/suite"
)

type ApplySuite struct {
	suite.Suite
	rootCmd *cobra.Command
}

//before all the suite
func (suite *ApplySuite) SetupAllSuite() {

}

//before each test
func (suite *ApplySuite) SetupTest() {
	system.InitConfigForTesting(suite.T())
	suite.rootCmd = RootCmd
}

func (suite *ApplySuite) TearDownTest() {

}

func (suite *ApplySuite) TearDownAllSuite() {

}

func (suite *ApplySuite) TestApplyJSON_GenericSubmit() {
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

			parsedBasedURI, _ := url.Parse(c.BaseURI)
			host, port, _ := net.SplitHostPort(parsedBasedURI.Host)
			_, out, err := ExecuteTestCobraCommand(suite.T(), suite.rootCmd, "apply",
				"--api-host", host,
				"--api-port", port,
				"-f", "../testdata/job.json",
			)
			assert.NoError(suite.T(), err, "Error submitting job. Run - Number of Jobs: %s. Job number: %s", tc.numberOfJobs, i)

			job, _, err := c.Get(ctx, strings.TrimSpace(out))
			assert.NoError(suite.T(), err)
			assert.NotNil(suite.T(), job, "Failed to get job with ID: %s", out)
		}()
	}
}

func (suite *ApplySuite) TestApplyYAML_GenericSubmit() {
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

			parsedBasedURI, _ := url.Parse(c.BaseURI)
			host, port, _ := net.SplitHostPort(parsedBasedURI.Host)
			_, out, err := ExecuteTestCobraCommand(suite.T(), suite.rootCmd, "apply",
				"--api-host", host,
				"--api-port", port,
				"-f", "../testdata/job.yaml",
			)
			assert.NoError(suite.T(), err, "Error submitting job. Run - Number of Jobs: %s. Job number: %s", tc.numberOfJobs, i)

			job, _, err := c.Get(ctx, strings.TrimSpace(out))
			assert.NoError(suite.T(), err)
			assert.NotNil(suite.T(), job, "Failed to get job with ID: %s", out)
		}()
	}
}

func TestApplySuite(t *testing.T) {
	suite.Run(t, new(RunSuite))
}
