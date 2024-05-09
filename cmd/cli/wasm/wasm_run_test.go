//go:build unit || !integration

package wasm_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"

	cmdtesting "github.com/bacalhau-project/bacalhau/cmd/testing"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	"github.com/bacalhau-project/bacalhau/pkg/storage/url/urldownload"
	testutils "github.com/bacalhau-project/bacalhau/pkg/test/utils"
)

type WasmRunSuite struct {
	cmdtesting.BaseSuite
}

func TestWasmRunSuite(t *testing.T) {
	suite.Run(t, new(WasmRunSuite))
}

func (s *WasmRunSuite) Test_SupportsRelativeDirectory() {
	ctx := context.Background()
	_, out, err := s.ExecuteTestCobraCommand("wasm", "run",
		"../../../testdata/wasm/noop/main.wasm",
	)
	s.Require().NoError(err)

	_ = testutils.GetJobFromTestOutput(ctx, s.T(), s.ClientV2, out)
}

func (s *WasmRunSuite) TestSpecifyingEnvVars() {
	ctx := context.Background()
	_, out, err := s.ExecuteTestCobraCommand("wasm", "run",
		"../../../testdata/wasm/env/main.wasm",
		"-e A=B,C=D",
	)
	s.Require().NoError(err)

	_ = testutils.GetJobFromTestOutput(ctx, s.T(), s.ClientV2, out)
}

func (s *WasmRunSuite) TestNoPublisher() {
	ctx := context.Background()
	_, out, err := s.ExecuteTestCobraCommand("wasm", "run",
		"../../../testdata/wasm/env/main.wasm",
	)
	s.Require().NoError(err)

	job := testutils.GetJobFromTestOutput(ctx, s.T(), s.ClientV2, out)
	info, err := s.ClientV2.Jobs().Get(ctx, &apimodels.GetJobRequest{
		JobID:   job.ID,
		Include: "executions",
	})
	s.Require().NoError(err)
	s.T().Log(info)

	s.Require().Len(info.Executions.Executions, 1)

	s.Require().Equal("", job.Task().Publisher.Type, "Expected empty publisher")
	s.Require().Empty(job.Task().Publisher.Params)
}

func (s *WasmRunSuite) TestLocalPublisher() {
	ctx := context.Background()
	_, out, err := s.ExecuteTestCobraCommand("wasm", "run",
		"-p", "local",
		"../../../testdata/wasm/env/main.wasm",
	)
	s.Require().NoError(err)

	job := testutils.GetJobFromTestOutput(ctx, s.T(), s.ClientV2, out)
	info, err := s.ClientV2.Jobs().Get(ctx, &apimodels.GetJobRequest{
		JobID:   job.ID,
		Include: "executions",
	})
	s.Require().NoError(err)
	s.T().Log(info)

	s.Require().Equal(models.PublisherLocal, job.Task().Publisher.Type, "Expected a local publisher")

	s.Require().Len(info.Executions.Executions, 1)

	exec := info.Executions.Executions[0]
	result := exec.PublishedResult
	s.Require().Equal(models.StorageSourceURL, result.Type)
	urlSpec, err := urldownload.DecodeSpec(result)
	s.Require().NoError(err)
	s.Require().Contains(urlSpec.URL, "http://127.0.0.1:", "URL does not contain expected prefix")
	s.Require().Contains(urlSpec.URL, fmt.Sprintf("%s.tgz", exec.ID), "URL does not contain expected file")
}
