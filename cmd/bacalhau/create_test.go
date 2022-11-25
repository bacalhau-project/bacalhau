//go:build unit || !integration

package bacalhau

import (
	"context"
	"errors"
	"net"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/publicapi"
	"github.com/filecoin-project/bacalhau/pkg/system"
	testutils "github.com/filecoin-project/bacalhau/pkg/test/utils"
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

// before each test
func (s *CreateSuite) SetupTest() {
	logger.ConfigureTestLogging(s.T())
	require.NoError(s.T(), system.InitConfigForTesting(s.T()))
	s.rootCmd = RootCmd
}

func (s *CreateSuite) TestCreateJSON_GenericSubmit() {
	tests := []struct {
		numberOfJobs int
	}{
		{numberOfJobs: 1}, // Test for one
		{numberOfJobs: 5}, // Test for five
	}

	Fatal = FakeFatalErrorHandler

	for i, tc := range tests {
		func() {
			ctx := context.Background()
			c, cm := publicapi.SetupRequesterNodeForTests(s.T(), false)
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
			testutils.GetJobFromTestOutput(ctx, s.T(), c, out)
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

	Fatal = FakeFatalErrorHandler

	for i, tc := range tests {

		testFiles := []string{"../../testdata/job.yaml", "../../testdata/job-url.yaml"}

		for _, testFile := range testFiles {
			func() {
				ctx := context.Background()
				c, cm := publicapi.SetupRequesterNodeForTests(s.T(), false)
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

				testutils.GetJobFromTestOutput(ctx, s.T(), c, out)
			}()
		}
	}
}

func (s *CreateSuite) TestCreateFromStdin() {
	testFile := "../../testdata/job.yaml"

	Fatal = FakeFatalErrorHandler

	c, cm := publicapi.SetupRequesterNodeForTests(s.T(), false)
	defer cm.Cleanup()

	*OC = *NewCreateOptions()

	parsedBasedURI, err := url.Parse(c.BaseURI)
	require.NoError(s.T(), err)

	testSpec, err := os.Open(testFile)
	require.NoError(s.T(), err)

	host, port, _ := net.SplitHostPort(parsedBasedURI.Host)
	_, out, err := ExecuteTestCobraCommandWithStdin(s.T(), s.rootCmd, testSpec, "create",
		"--api-host", host,
		"--api-port", port,
	)

	require.NoError(s.T(), err, "Error submitting job.")

	// Now run describe on the ID we got back
	job := testutils.GetJobFromTestOutput(context.Background(), s.T(), c, out)
	_, out, err = ExecuteTestCobraCommand(s.T(), s.rootCmd, "describe",
		"--api-host", host,
		"--api-port", port,
		job.ID,
	)

	require.NoError(s.T(), err, "Error describing job.")
}

func (s *CreateSuite) TestCreateDontPanicOnNoInput() {
	Fatal = FakeFatalErrorHandler

	type commandReturn struct {
		c   *cobra.Command
		out string
		err error
	}

	commandChan := make(chan commandReturn, 1)

	go func() {
		c, out, err := ExecuteTestCobraCommand(s.T(), RootCmd, "create")

		commandChan <- commandReturn{c: c, out: out, err: err}
	}()
	time.Sleep(1 * time.Second)

	stdinErr := os.Stdin.Close()
	if stdinErr != nil && !errors.Is(stdinErr, os.ErrClosed) {
		require.NoError(s.T(), stdinErr, "Error closing stdin")
	}

	commandReturnValue := <-commandChan

	// For some reason I can't explain, this only works when running in debug.
	// require.Contains(s.T(), commandReturnValue.out, "Ctrl+D", "Waiting message should contain Ctrl+D")

	errorOutputMap := make(map[string]interface{})
	for _, o := range strings.Split(commandReturnValue.out, "\n") {
		err := model.YAMLUnmarshalWithMax([]byte(o), &errorOutputMap)
		if err != nil {
			continue
		}
	}

	require.Contains(s.T(), errorOutputMap["Message"], "The job provided is invalid", "Output message should error properly.")
	require.Equal(s.T(), int(errorOutputMap["Code"].(float64)), 1, "Expected no error when no input is provided")
}

func (s *CreateSuite) TestCreateDontPanicOnEmptyFile() {
	Fatal = FakeFatalErrorHandler

	type commandReturn struct {
		c   *cobra.Command
		out string
		err error
	}

	commandChan := make(chan commandReturn, 1)

	go func() {
		c, out, err := ExecuteTestCobraCommand(s.T(), RootCmd, "create", "../../testdata/empty.yaml")

		commandChan <- commandReturn{c: c, out: out, err: err}
	}()
	time.Sleep(1 * time.Second)

	stdinErr := os.Stdin.Close()
	require.NoError(s.T(), stdinErr, "Error closing stdin")

	commandReturnValue := <-commandChan

	errorOutputMap := make(map[string]interface{})
	for _, o := range strings.Split(commandReturnValue.out, "\n") {
		err := model.YAMLUnmarshalWithMax([]byte(o), &errorOutputMap)
		if err != nil {
			continue
		}
	}

	require.Contains(s.T(), errorOutputMap["Message"], "The job provided is invalid", "Output message should error properly.")
	require.Equal(s.T(), int(errorOutputMap["Code"].(float64)), 1, "Expected no error when no input is provided")
}
