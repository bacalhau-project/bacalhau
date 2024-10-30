//go:build unit || !integration

package logstream_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"
	"github.com/vincent-petithory/dataurl"

	"github.com/bacalhau-project/bacalhau/pkg/docker"
	dockermodels "github.com/bacalhau-project/bacalhau/pkg/executor/docker/models"
	wasmmodels "github.com/bacalhau-project/bacalhau/pkg/executor/wasm/models"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	clientv2 "github.com/bacalhau-project/bacalhau/pkg/publicapi/client/v2"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
	"github.com/bacalhau-project/bacalhau/pkg/storage/inline"
	"github.com/bacalhau-project/bacalhau/pkg/test/scenario"
	"github.com/bacalhau-project/bacalhau/testdata/wasm/cat"

	"github.com/bacalhau-project/bacalhau/pkg/devstack"
	testutil "github.com/bacalhau-project/bacalhau/pkg/test/teststack"
)

// LogStreamTestSuite tests the log streaming functionality for different execution engines.
// It verifies that logs can be retrieved through the public API while jobs are running.
type LogStreamTestSuite struct {
	suite.Suite

	ctx           context.Context         // Context for the test suite
	stack         *devstack.DevStack      // Local test stack
	client        clientv2.API            // Client for the API server
	stateResolver *scenario.StateResolver // Helper for checking job states
}

func TestLogStreamTestSuite(t *testing.T) {
	suite.Run(t, new(LogStreamTestSuite))
}

// SetupSuite initializes the test environment with a single hybrid node
func (s *LogStreamTestSuite) SetupSuite() {
	s.ctx = context.Background()
	s.stack = testutil.Setup(s.ctx, s.T(), devstack.WithNumberOfHybridNodes(1))

	apiServer := s.stack.Nodes[0].APIServer
	s.client = clientv2.New(fmt.Sprintf("http://%s:%d", apiServer.Address, apiServer.Port))
	s.stateResolver = scenario.NewStateResolverFromAPI(s.client)
}

// runLogStreamTest runs a standardized test for log streaming:
// 1. Submits a job
// 2. Waits for it to start running
// 3. Verifies log output can be retrieved
// 4. Cleans up the job
func (s *LogStreamTestSuite) runLogStreamTest(job *models.Job) {
	ctx, cancelFunc := context.WithTimeout(s.ctx, time.Second*10)
	defer cancelFunc()

	// Submit the job
	resp, err := s.client.Jobs().Put(ctx, &apimodels.PutJobRequest{
		Job: job,
	})
	s.Require().NoError(err)

	// Ensure job is cleaned up after test
	defer func() {
		_, err = s.client.Jobs().Stop(ctx, &apimodels.StopJobRequest{
			JobID: resp.JobID,
		})
		s.Require().NoError(err)
	}()

	// Wait for job to start running
	s.Require().NoError(s.stateResolver.Wait(ctx, resp.JobID, scenario.WaitForRunningState()))

	// Get log stream
	ch, err := s.client.Jobs().Logs(ctx, &apimodels.GetLogsRequest{
		JobID:  resp.JobID,
		Follow: true,
		Tail:   true,
	})
	s.Require().NoError(err)
	s.Require().NotNil(ch)

	// Verify log output
	select {
	case asyncResult, ok := <-ch:
		s.Require().True(ok)
		s.Require().NoError(asyncResult.Err)
		s.Require().Equal(models.ExecutionLogTypeSTDOUT, asyncResult.Value.Type)
		s.Require().Contains(asyncResult.Value.Line, "logstreamoutput")
	case <-ctx.Done():
		s.Require().Fail("timed out waiting for log stream")
	}
}

// TestDockerOutputStream verifies log streaming works for Docker-based jobs
func (s *LogStreamTestSuite) TestDockerOutputStream() {
	docker.MustHaveDocker(s.T())
	job := &models.Job{
		Type:  models.JobTypeBatch,
		Count: 1,
		Tasks: []*models.Task{
			{
				Name: "task1",
				Engine: &models.SpecConfig{
					Type: models.EngineDocker,
					Params: dockermodels.EngineSpec{
						Image:      "busybox:latest",
						Entrypoint: []string{"sh", "-c", "for i in {1..100}; do echo \"logstreamoutput\"; sleep .1; done"},
					}.ToMap(),
				},
			},
		},
	}
	s.runLogStreamTest(job)
}

// TestWasmOutputStream verifies log streaming works for WASM-based jobs
func (s *LogStreamTestSuite) TestWasmOutputStream() {
	s.T().Skip("https://github.com/bacalhau-project/bacalhau/issues/4158")
	job := &models.Job{
		Type:  models.JobTypeBatch,
		Count: 1,
		Tasks: []*models.Task{
			{
				Name: "task1",
				Engine: &models.SpecConfig{
					Type: models.EngineWasm,
					Params: wasmmodels.EngineArguments{
						EntryModule: storage.PreparedStorage{
							InputSource: models.InputSource{
								Source: &models.SpecConfig{
									Type: models.StorageSourceInline,
									Params: inline.Source{
										URL: dataurl.EncodeBytes(cat.Program()),
									}.ToMap(),
								},
							}},
						EntryPoint: "_start",
					}.ToMap(),
				},
			},
		},
	}
	s.runLogStreamTest(job)
}
