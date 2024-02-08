//go:build unit || !integration

package create_test

import (
	"context"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/lib/marshaller"
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
		Fixture *testdata.FixtureLegacy
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
				FixtureLegacy: testdata.TaskWasmJson,
			},
		*/
		{
			Name:    "docker engine spec json",
			Fixture: testdata.JsonJobDockerEngineSpec,
		},
		{
			Name:    "docker engine spec yaml",
			Fixture: testdata.YamlJobDockerEngineSpec,
		},
		{
			Name:    "wasm engine spec json",
			Fixture: testdata.JsonJobWasmEngineSpec,
		},
	}

	// Let's do this once
	canRunS3Test := s3helper.CanRunS3Test()

	for _, tc := range tests {
		s.Run(tc.Name, func() {
			if tc.Fixture.RequiresS3() && !canRunS3Test {
				// Skip the S3 tests if we have no AWS credentials installed
				s.T().Skip("No valid AWS credentials found")
			}

			if tc.Fixture.RequiresDocker() {
				docker.MustHaveDocker(s.T())
			}

			ctx := context.Background()
			_, out, err := s.ExecuteTestCobraCommandWithStdinBytes(tc.Fixture.Data, "create")

			require.NoError(s.T(), err, "Error submitting job")
			testutils.GetJobFromTestOutputLegacy(ctx, s.T(), s.Client, out)
		})
	}
}

func (s *CreateSuite) TestCreateFromStdin() {
	_, out, err := s.ExecuteTestCobraCommandWithStdinBytes(testdata.YamlJobNoop.Data, "create")

	require.NoError(s.T(), err, "Error submitting job.")

	// Now run describe on the ID we got back
	job := testutils.GetJobFromTestOutputLegacy(context.Background(), s.T(), s.Client, out)
	_, _, err = s.ExecuteTestCobraCommand("describe", job.Metadata.ID)

	require.NoError(s.T(), err, "Error describing job.")
}

// cspell:ignore Dont
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
		c, out, err := s.ExecuteTestCobraCommand("create", "./testdata/empty.yaml")

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

	require.Contains(s.T(), errorOutputMap["Message"], "the job provided is invalid", "Output message should error properly.")
	require.Equal(s.T(), int(errorOutputMap["Code"].(float64)), 1, "Expected no error when no input is provided")
}
