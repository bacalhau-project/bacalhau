//go:build unit || !integration

package create_test

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"

	cmdtesting "github.com/bacalhau-project/bacalhau/cmd/testing"
	"github.com/bacalhau-project/bacalhau/pkg/docker"
	s3helper "github.com/bacalhau-project/bacalhau/pkg/s3"
	testutils "github.com/bacalhau-project/bacalhau/pkg/test/utils"
	"github.com/bacalhau-project/bacalhau/testdata"
)

type CreateSuite struct {
	cmdtesting.BaseSuite
}

func TestCreateSuite(t *testing.T) {
	suite.Run(t, new(CreateSuite))
}

func (s *CreateSuite) TestCreateGenericSubmitBetter() {
	tests := []struct {
		Name    string
		Fixture *testdata.Fixture
	}{
		{
			Name:    "noop json",
			Fixture: testdata.JsonJobNoop,
		},
		{
			Name:    "noop yaml",
			Fixture: testdata.YamlJobNoop,
		},
		{
			Name:    "s3 yaml",
			Fixture: testdata.YamlJobS3,
		},
		{
			Name:    "url noop yaml",
			Fixture: testdata.YamlJobNoopUrl,
		},
		{
			Name:    "docker task json",
			Fixture: testdata.IPVMTaskDocker,
		},
		{
			Name:    "task with config json",
			Fixture: testdata.IPVMTaskWithConfig,
		},
		//"TODO: re-enable wasm job which is currently broken as it relies on pulling data from the public IPFS network")
		/*
			{
				Name:    "wasm task json",
				Fixture: testdata.TaskWasmJson,
			},
		*/
	}

	for _, tc := range tests {
		s.Run(tc.Name, func() {
			if tc.Fixture.RequiresS3() && !s3helper.CanRunS3Test() {
				// Skip the S3 tests if we have no AWS credentials installed
				s.T().Skip("No valid AWS credentials found")
			}

			if tc.Fixture.RequiresDocker() {
				docker.MustHaveDocker(s.T())
			}

			ctx := context.Background()
			_, out, err := cmdtesting.ExecuteTestCobraCommandWithStdinBytes(tc.Fixture.Data, "create",
				"--api-host", s.Host,
				"--api-port", fmt.Sprint(s.Port),
			)

			fmt.Println(tc.Fixture.Data)

			require.NoError(s.T(), err, "Error submitting job")
			testutils.GetJobFromTestOutput(ctx, s.T(), s.Client, out)
		})
	}
}

func (s *CreateSuite) TestCreateFromStdin() {
	_, out, err := cmdtesting.ExecuteTestCobraCommandWithStdinBytes(testdata.YamlJobNoop.Data, "create",
		"--api-host", s.Host,
		"--api-port", fmt.Sprint(s.Port),
	)

	require.NoError(s.T(), err, "Error submitting job.")

	// Now run describe on the ID we got back
	job := testutils.GetJobFromTestOutput(context.Background(), s.T(), s.Client, out)
	_, _, err = cmdtesting.ExecuteTestCobraCommand("describe",
		"--api-host", s.Host,
		"--api-port", fmt.Sprint(s.Port),
		job.Metadata.ID,
	)

	require.NoError(s.T(), err, "Error describing job.")
}

func (s *CreateSuite) TestCreateDontPanicOnEmptyFile() {
	type commandReturn struct {
		c   *cobra.Command
		out string
		err error
	}

	commandChan := make(chan commandReturn, 1)

	wg := sync.WaitGroup{}
	wg.Add(1)
	go func() {
		defer wg.Done()
		c, out, err := cmdtesting.ExecuteTestCobraCommand("create", "./testdata/empty.yaml")

		commandChan <- commandReturn{c: c, out: out, err: err}
	}()
	wg.Wait()

	stdinErr := os.Stdin.Close()
	require.NoError(s.T(), stdinErr, "Error closing stdin")

	commandReturnValue := <-commandChan

	errorOutputMap := make(map[string]interface{})
	for _, o := range strings.Split(commandReturnValue.out, "\n") {
		err := marshaller.YAMLUnmarshalWithMax([]byte(o), &errorOutputMap)
		if err != nil {
			continue
		}
	}

	require.Contains(s.T(), errorOutputMap["Message"], "The job provided is invalid", "Output message should error properly.")
	require.Equal(s.T(), int(errorOutputMap["Code"].(float64)), 1, "Expected no error when no input is provided")
}
