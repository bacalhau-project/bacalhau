//go:build unit || !integration

package wasm_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

	cmdtesting "github.com/bacalhau-project/bacalhau/cmd/testing"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/test/scenario"
	testutils "github.com/bacalhau-project/bacalhau/pkg/test/utils"
)

type WasmRunSuite struct {
	cmdtesting.BaseSuite
}

func TestWasmRunSuite(t *testing.T) {
	suite.Run(t, new(WasmRunSuite))
}

func (s *WasmRunSuite) TestRelativeLocalFileInputSource() {
	ctx := context.Background()
	_, out, err := s.ExecuteTestCobraCommand("wasm", "run",
		"../../../testdata/wasm/noop/main.wasm",
	)
	s.Require().NoError(err)

	job := testutils.GetJobFromTestOutput(ctx, s.T(), s.ClientV2, out)

	// Create a state resolver to wait for job completion
	stateResolver := scenario.NewStateResolverFromAPI(s.ClientV2)

	// Wait for the job to complete successfully
	err = stateResolver.Wait(ctx, job.ID, scenario.WaitForSuccessfulCompletion())
	s.Require().NoError(err, "Job should have completed successfully")
}

func (s *WasmRunSuite) TestSpecifyingEnvVars() {
	ctx := context.Background()
	_, out, err := s.ExecuteTestCobraCommand("wasm", "run",
		"../../../testdata/wasm/env/main.wasm",
		"-e", "A=B,C=D",
	)
	s.Require().NoError(err)

	job := testutils.GetJobFromTestOutput(ctx, s.T(), s.ClientV2, out)

	// Create a state resolver to wait for job completion
	stateResolver := scenario.NewStateResolverFromAPI(s.ClientV2)

	// Wait for the job to complete successfully
	err = stateResolver.Wait(ctx, job.ID, scenario.WaitForSuccessfulCompletion())
	s.Require().NoError(err, "Job should have completed successfully")
}

func (s *WasmRunSuite) TestLocalFileInputSource() {
	ctx := context.Background()
	_, out, err := s.ExecuteTestCobraCommand("wasm", "run",
		"../../../testdata/wasm/noop/main.wasm",
	)
	s.Require().NoError(err)

	job := testutils.GetJobFromTestOutput(ctx, s.T(), s.ClientV2, out)
	s.Require().Equal("main.wasm", job.Task().Engine.Params["EntryModule"])

	// Verify input source was added
	s.Require().Len(job.Task().InputSources, 1)
	s.Require().Equal(models.StorageSourceInline, job.Task().InputSources[0].Source.Type)
	s.Require().Equal("main.wasm", job.Task().InputSources[0].Target)

	// Create a state resolver to wait for job completion
	stateResolver := scenario.NewStateResolverFromAPI(s.ClientV2)

	// Wait for the job to complete successfully
	err = stateResolver.Wait(ctx, job.ID, scenario.WaitForSuccessfulCompletion())
	s.Require().NoError(err, "Job should have completed successfully")
}

func (s *WasmRunSuite) TestLocalFileWithCustomTarget() {
	ctx := context.Background()
	_, out, err := s.ExecuteTestCobraCommand("wasm", "run",
		"../../../testdata/wasm/noop/main.wasm:/app/custom.wasm",
	)
	s.Require().NoError(err)

	job := testutils.GetJobFromTestOutput(ctx, s.T(), s.ClientV2, out)
	s.Require().Equal("/app/custom.wasm", job.Task().Engine.Params["EntryModule"])

	// Verify input source was added
	s.Require().Len(job.Task().InputSources, 1)
	s.Require().Equal(models.StorageSourceInline, job.Task().InputSources[0].Source.Type)
	s.Require().Equal("/app/custom.wasm", job.Task().InputSources[0].Target)

	// Create a state resolver to wait for job completion
	stateResolver := scenario.NewStateResolverFromAPI(s.ClientV2)

	// Wait for the job to complete successfully
	err = stateResolver.Wait(ctx, job.ID, scenario.WaitForSuccessfulCompletion())
	s.Require().NoError(err, "Job should have completed successfully")
}
