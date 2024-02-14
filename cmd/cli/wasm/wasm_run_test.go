//go:build unit || !integration

package wasm_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/suite"

	cmdtesting "github.com/bacalhau-project/bacalhau/cmd/testing"
	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	testutils "github.com/bacalhau-project/bacalhau/pkg/test/utils"
)

type WasmRunSuite struct {
	cmdtesting.BaseSuite
}

func TestWasmRunSuite(t *testing.T) {
	util.Fatal = util.FakeFatalErrorHandler
	suite.Run(t, new(WasmRunSuite))
}

func (s *WasmRunSuite) Test_SupportsRelativeDirectory() {
	ctx := context.Background()
	_, out, err := s.ExecuteTestCobraCommand("wasm", "run",
		"../../../testdata/wasm/noop/main.wasm",
	)
	s.Require().NoError(err)

	_ = testutils.GetJobFromTestOutputLegacy(ctx, s.T(), s.Client, out)
}

func (s *WasmRunSuite) TestSpecifyingEnvVars() {
	ctx := context.Background()
	_, out, err := s.ExecuteTestCobraCommand("wasm", "run",
		"../../../testdata/wasm/env/main.wasm",
		"-e A=B,C=D",
	)
	s.Require().NoError(err)

	_ = testutils.GetJobFromTestOutputLegacy(ctx, s.T(), s.Client, out)
}

func (s *WasmRunSuite) TestNoPublisher() {
	ctx := context.Background()
	_, out, err := s.ExecuteTestCobraCommand("wasm", "run",
		"../../../testdata/wasm/env/main.wasm",
		"-e A=B,C=D",
	)
	s.Require().NoError(err)

	job := testutils.GetJobFromTestOutputLegacy(ctx, s.T(), s.Client, out)
	info, _, err := s.Client.Get(ctx, job.Metadata.ID)
	s.Require().NoError(err)
	s.T().Log(info)

	s.Require().Len(info.State.Executions, 1)

	exec := info.State.Executions[0]
	result := exec.PublishedResult

	s.Require().Equal("noop", job.Spec.PublisherSpec.Type.String(), "Expected a noop publisher")
	s.Require().Empty(result.URL, "Did not expect a URL")
	s.Require().Empty(result.S3, "Did not expect S3 details")
	s.Require().Empty(result.CID, "Did not expect a CID")
}

func (s *WasmRunSuite) TestLocalPublisher() {
	ctx := context.Background()
	_, out, err := s.ExecuteTestCobraCommand("wasm", "run",
		"-p", "local",
		"../../../testdata/wasm/env/main.wasm",
		"-e A=B,C=D",
	)
	s.Require().NoError(err)

	job := testutils.GetJobFromTestOutputLegacy(ctx, s.T(), s.Client, out)
	info, _, err := s.Client.Get(ctx, job.Metadata.ID)
	s.Require().NoError(err)
	s.T().Log(info)

	s.Require().Equal(model.PublisherLocal, job.Spec.PublisherSpec.Type, "Expected a local publisher")

	s.Require().Len(info.State.Executions, 1)

	exec := info.State.Executions[0]
	result := exec.PublishedResult
	s.Require().Equal(model.StorageSourceURLDownload, result.StorageSource)
	s.Require().Contains(result.URL, "http://127.0.0.1:", "URL does not contain expected prefix")
	s.Require().Contains(result.URL, fmt.Sprintf("%s.tgz", exec.ID().ExecutionID), "URL does not contain expected file")
}
