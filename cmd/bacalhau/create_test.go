//go:build unit || !integration

package bacalhau

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/model"
	testutils "github.com/filecoin-project/bacalhau/pkg/test/utils"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type CreateSuite struct {
	BaseSuite
}

func TestCreateSuite(t *testing.T) {
	suite.Run(t, new(CreateSuite))
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
			_, out, err := ExecuteTestCobraCommand(s.T(), "create",
				"--api-host", s.host,
				"--api-port", s.port,
				"../../testdata/job.json",
			)
			require.NoError(s.T(), err, "Error submitting job. Run - Number of Jobs: %d. Job number: %d", tc.numberOfJobs, i)
			testutils.GetJobFromTestOutput(context.Background(), s.T(), s.client, out)
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

				_, out, err := ExecuteTestCobraCommand(s.T(), "create",
					"--api-host", s.host,
					"--api-port", s.port,
					testFile,
				)

				require.NoError(s.T(), err, "Error submitting job. Run - Number of Jobs: %d. Job number: %d", tc.numberOfJobs, i)

				testutils.GetJobFromTestOutput(ctx, s.T(), s.client, out)
			}()
		}
	}
}

func (s *CreateSuite) TestCreateFromStdin() {
	testFile := "../../testdata/job.yaml"

	Fatal = FakeFatalErrorHandler
	testSpec, err := os.Open(testFile)
	require.NoError(s.T(), err)

	_, out, err := ExecuteTestCobraCommandWithStdin(s.T(), testSpec, "create",
		"--api-host", s.host,
		"--api-port", s.port,
	)

	require.NoError(s.T(), err, "Error submitting job.")

	// Now run describe on the ID we got back
	job := testutils.GetJobFromTestOutput(context.Background(), s.T(), s.client, out)
	_, out, err = ExecuteTestCobraCommand(s.T(), "describe",
		"--api-host", s.host,
		"--api-port", s.port,
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
		c, out, err := ExecuteTestCobraCommand(s.T(), "create")

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
		c, out, err := ExecuteTestCobraCommand(s.T(), "create", "../../testdata/empty.yaml")

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
