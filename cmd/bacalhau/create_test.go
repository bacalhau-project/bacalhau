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

// before all the s
func (s *CreateSuite) Setups() {

}

// before each test
func (s *CreateSuite) SetupTest() {
	require.NoError(s.T(), system.InitConfigForTesting())
	s.rootCmd = RootCmd
}

func (s *CreateSuite) TearDownTest() {

}

func (s *CreateSuite) TearDownAlls() {

}

func (s *CreateSuite) TestCreateJSON_GenericSubmit() {
	tests := []struct {
		numberOfJobs int
	}{
		{numberOfJobs: 1}, // Test for one
		{numberOfJobs: 5}, // Test for five
	}

	for i, tc := range tests {
		func() {
			ctx := context.Background()
			c, cm := publicapi.SetupTests(s.T())
			defer cm.Cleanup()

			*OC = *NewCreateOptions()

			parsedBasedURI, err := url.Parse(c.BaseURI)
			require.NoError(s.T(), err)

			host, port, _ := net.SplitHostPort(parsedBasedURI.Host)
			_, out, err := ExecuteTestCobraCommand(s.T(), s.rootCmd, "create",
				"--api-host", host,
				"--api-port", port,
				"../../testdata/job.json",
			)
			require.NoError(s.T(), err, "Error submitting job. Run - Number of Jobs: %d. Job number: %d", tc.numberOfJobs, i)

			job, _, err := c.Get(ctx, strings.TrimSpace(out))
			require.NoError(s.T(), err)
			require.NotNil(s.T(), job, "Failed to get job with ID: %s", out)
		}()
	}
}

func (s *CreateSuite) TestCreateYAML_GenericSubmit() {
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
				c, cm := publicapi.SetupTests(s.T())
				defer cm.Cleanup()

				*OC = *NewCreateOptions()

				parsedBasedURI, err := url.Parse(c.BaseURI)
				require.NoError(s.T(), err)

				host, port, _ := net.SplitHostPort(parsedBasedURI.Host)
				_, out, err := ExecuteTestCobraCommand(s.T(), s.rootCmd, "create",
					"--api-host", host,
					"--api-port", port,
					testFile,
				)

				require.NoError(s.T(), err, "Error submitting job. Run - Number of Jobs: %d. Job number: %d", tc.numberOfJobs, i)

				job, _, err := c.Get(ctx, strings.TrimSpace(out))
				require.NoError(s.T(), err)
				require.NotNil(s.T(), job, "Failed to get job with ID: %s", out)
			}()
		}
	}
}

func (s *CreateSuite) TestCreateFromStdin() {
	testFile := "../../testdata/job-url.yaml"

	c, cm := publicapi.SetupTests(s.T())
	defer cm.Cleanup()

	*OC = *NewCreateOptions()

	parsedBasedURI, err := url.Parse(c.BaseURI)
	require.NoError(s.T(), err)

	host, port, _ := net.SplitHostPort(parsedBasedURI.Host)
	_, out, err := ExecuteTestCobraCommand(s.T(), s.rootCmd, "create",
		"--api-host", host,
		"--api-port", port,
		testFile,
	)

	require.NoError(s.T(), err, "Error submitting job.")

	// Now run describe on the ID we got back
	_, out, err = ExecuteTestCobraCommand(s.T(), s.rootCmd, "describe",
		"--api-host", host,
		"--api-port", port,
		strings.TrimSpace(out),
	)

	require.NoError(s.T(), err, "Error describing job.")

	// Cat the file and pipe it to stdin
	r, err := system.UnsafeForUserCodeRunCommand( //nolint:govet // shadowing ok
		"echo", []string{out,
			"|", "../../bin/bacalhau create -"},
	)
	require.Equal(s.T(), 0, r.ExitCode, "Error piping to stdin")
}
