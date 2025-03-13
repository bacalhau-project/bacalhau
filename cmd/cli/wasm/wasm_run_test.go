//go:build unit || !integration

package wasm_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/suite"

	cmdtesting "github.com/bacalhau-project/bacalhau/cmd/testing"
	"github.com/bacalhau-project/bacalhau/pkg/models"
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

	_ = testutils.GetJobFromTestOutput(ctx, s.T(), s.ClientV2, out)
}

func (s *WasmRunSuite) TestSpecifyingEnvVars() {
	ctx := context.Background()
	_, out, err := s.ExecuteTestCobraCommand("wasm", "run",
		"../../../testdata/wasm/env/main.wasm",
		"-e", "A=B,C=D",
	)
	s.Require().NoError(err)

	_ = testutils.GetJobFromTestOutput(ctx, s.T(), s.ClientV2, out)
}

func (s *WasmRunSuite) TestStorageSpecWithCustomTarget() {
	_, out, err := s.ExecuteTestCobraCommand("wasm", "run",
		"s3://bucket/main.wasm:/app/custom.wasm",
	)
	s.Require().NoError(err)

	job := testutils.GetJobFromTestOutput(context.Background(), s.T(), s.ClientV2, out)
	s.Require().Equal("/app/custom.wasm", job.Task().Engine.Params["EntryModule"])

	// Verify input source was added
	s.Require().Len(job.Task().InputSources, 1)
	s.Require().Equal(models.StorageSourceS3, job.Task().InputSources[0].Source.Type)
	s.Require().Equal("bucket", job.Task().InputSources[0].Source.Params["Bucket"])
	s.Require().Equal("main.wasm", job.Task().InputSources[0].Source.Params["Key"])
	s.Require().Equal("/app/custom.wasm", job.Task().InputSources[0].Target)
}

func (s *WasmRunSuite) TestLocalFileInputSource() {
	_, out, err := s.ExecuteTestCobraCommand("wasm", "run",
		"../../../testdata/wasm/noop/main.wasm",
	)
	s.Require().NoError(err)

	job := testutils.GetJobFromTestOutput(context.Background(), s.T(), s.ClientV2, out)
	s.Require().Equal("main.wasm", job.Task().Engine.Params["EntryModule"])

	// Verify input source was added
	s.Require().Len(job.Task().InputSources, 1)
	s.Require().Equal(models.StorageSourceInline, job.Task().InputSources[0].Source.Type)
	s.Require().Equal("main.wasm", job.Task().InputSources[0].Target)
}

func (s *WasmRunSuite) TestLocalFileWithCustomTarget() {
	_, out, err := s.ExecuteTestCobraCommand("wasm", "run",
		"../../../testdata/wasm/noop/main.wasm:/app/custom.wasm",
	)
	s.Require().NoError(err)

	job := testutils.GetJobFromTestOutput(context.Background(), s.T(), s.ClientV2, out)
	s.Require().Equal("/app/custom.wasm", job.Task().Engine.Params["EntryModule"])

	// Verify input source was added
	s.Require().Len(job.Task().InputSources, 1)
	s.Require().Equal(models.StorageSourceInline, job.Task().InputSources[0].Source.Type)
	s.Require().Equal("/app/custom.wasm", job.Task().InputSources[0].Target)
}

func (s *WasmRunSuite) TestImportModules() {
	_, out, err := s.ExecuteTestCobraCommand("wasm", "run",
		"/app/main.wasm",
		"--import-modules", "/app/lib.wasm",
		"--input", "s3://bucket/entry/main.wasm:/app/main.wasm",
		"--input", "s3://bucket/module/main.wasm:/app/lib.wasm",
	)
	s.Require().NoError(err)

	job := testutils.GetJobFromTestOutput(context.Background(), s.T(), s.ClientV2, out)
	s.Require().Equal("/app/main.wasm", job.Task().Engine.Params["EntryModule"])
	s.Require().Equal([]interface{}{"/app/lib.wasm"}, job.Task().Engine.Params["ImportModules"])

	// Verify input sources were added
	s.Require().Len(job.Task().InputSources, 2)
	s.Require().Equal(models.StorageSourceS3, job.Task().InputSources[0].Source.Type)
	s.Require().Equal("bucket", job.Task().InputSources[0].Source.Params["Bucket"])
	s.Require().Equal("entry/main.wasm", job.Task().InputSources[0].Source.Params["Key"])
	s.Require().Equal("/app/main.wasm", job.Task().InputSources[0].Target)

	s.Require().Equal(models.StorageSourceS3, job.Task().InputSources[1].Source.Type)
	s.Require().Equal("bucket", job.Task().InputSources[1].Source.Params["Bucket"])
	s.Require().Equal("module/main.wasm", job.Task().InputSources[1].Source.Params["Key"])
	s.Require().Equal("/app/lib.wasm", job.Task().InputSources[1].Target)
}

func (s *WasmRunSuite) TestImportModulesWithStorageSpecs() {
	_, out, err := s.ExecuteTestCobraCommand("wasm", "run",
		"/app/main.wasm",
		"--import-modules", "s3://bucket/lib1.wasm:/app/lib1.wasm",
		"--import-modules", "s3://bucket/lib2.wasm:/app/lib2.wasm",
		"--input", "s3://bucket/entry/main.wasm:/app/main.wasm",
	)
	s.Require().NoError(err)

	job := testutils.GetJobFromTestOutput(context.Background(), s.T(), s.ClientV2, out)
	s.Require().Equal("/app/main.wasm", job.Task().Engine.Params["EntryModule"])
	s.Require().Equal([]interface{}{"/app/lib1.wasm", "/app/lib2.wasm"}, job.Task().Engine.Params["ImportModules"])

	// Verify input sources were added
	s.Require().Len(job.Task().InputSources, 3)

	// Check entry module
	s.Require().Equal(models.StorageSourceS3, job.Task().InputSources[0].Source.Type)
	s.Require().Equal("bucket", job.Task().InputSources[0].Source.Params["Bucket"])
	s.Require().Equal("entry/main.wasm", job.Task().InputSources[0].Source.Params["Key"])
	s.Require().Equal("/app/main.wasm", job.Task().InputSources[0].Target)

	// Check first import module
	s.Require().Equal(models.StorageSourceS3, job.Task().InputSources[1].Source.Type)
	s.Require().Equal("bucket", job.Task().InputSources[1].Source.Params["Bucket"])
	s.Require().Equal("lib1.wasm", job.Task().InputSources[1].Source.Params["Key"])
	s.Require().Equal("/app/lib1.wasm", job.Task().InputSources[1].Target)

	// Check second import module
	s.Require().Equal(models.StorageSourceS3, job.Task().InputSources[2].Source.Type)
	s.Require().Equal("bucket", job.Task().InputSources[2].Source.Params["Bucket"])
	s.Require().Equal("lib2.wasm", job.Task().InputSources[2].Source.Params["Key"])
	s.Require().Equal("/app/lib2.wasm", job.Task().InputSources[2].Target)
}
