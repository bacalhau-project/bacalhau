//go:build unit || !integration

package logstream_test

import (
	"context"
	"time"

	"github.com/stretchr/testify/require"
	"github.com/vincent-petithory/dataurl"

	"github.com/bacalhau-project/bacalhau/pkg/executor"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
	"github.com/bacalhau-project/bacalhau/pkg/storage/inline"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/bacalhau-project/bacalhau/pkg/test/mock"
	"github.com/bacalhau-project/bacalhau/testdata/wasm/cat"

	wasmmodels "github.com/bacalhau-project/bacalhau/pkg/executor/wasm/models"
)

func (s *LogStreamTestSuite) TestWasmOutputStream() {
	s.T().Skip("https://github.com/bacalhau-project/bacalhau/issues/4158")
	node := s.stack.Nodes[0]

	ctx, cancelFunc := context.WithTimeout(s.ctx, time.Duration(10)*time.Second)
	defer cancelFunc()

	success := make(chan bool, 1)
	fail := make(chan bool, 1)

	task := mock.TaskBuilder().
		Engine(
			&models.SpecConfig{
				Type: models.EngineWasm,
				Params: wasmmodels.EngineArguments{
					EntryModule: storage.PreparedStorage{
						InputSource: models.InputSource{
							Target: "/inputs",
							Source: &models.SpecConfig{
								Type: models.StorageSourceInline,
								Params: inline.Source{
									URL: dataurl.EncodeBytes(cat.Program()),
								}.ToMap(),
							},
						}},
					EntryPoint: "_start",
				}.ToMap(),
			}).
		BuildOrDie()
	job := mock.Job()
	job.Tasks[0] = task
	job.Normalize()

	execution := mock.ExecutionForJob(job)
	execution.AllocateResources(task.Name, models.Resources{})

	err := node.RequesterNode.JobStore.CreateJob(s.ctx, *job, models.Event{})
	require.NoError(s.T(), err)

	exec, err := node.ComputeNode.Executors.Get(s.ctx, models.EngineWasm)
	require.NoError(s.T(), err)

	go func() {
		// Run the job.  We won't ever get a result because of the
		// entrypoint we chose, but we might get timed-out.
		_, err = exec.Run(
			ctx,
			&executor.RunCommandRequest{
				JobID:        job.ID,
				ExecutionID:  execution.ID,
				Resources:    execution.TotalAllocatedResources(),
				Network:      task.Network,
				Outputs:      task.ResultPaths,
				Inputs:       nil,
				ResultsDir:   "/tmp",
				EngineParams: task.Engine,
				OutputLimits: executor.OutputLimits{
					MaxStdoutFileLength:   system.MaxStdoutFileLength,
					MaxStdoutReturnLength: system.MaxStdoutReturnLength,
					MaxStderrFileLength:   system.MaxStderrFileLength,
					MaxStderrReturnLength: system.MaxStderrReturnLength,
				},
			},
		)
		if err != nil {
			s.T().Log(err)
			fail <- true
		}
		fail <- true
	}()

	go func() {
		// TODO(walid): [correctness] following the recommendation in docker test to wait for module to run
		time.Sleep(time.Second * 3)
		ch, err := waitForOutputStream(ctx, execution.ID, true, true, exec)
		require.NoError(s.T(), err)
		require.NotNil(s.T(), ch)

		asyncResult, ok := <-ch
		require.True(s.T(), ok)
		require.NoError(s.T(), asyncResult.Err)
		require.Equal(s.T(), models.ExecutionLogTypeSTDOUT, asyncResult.Value.Type)
		require.Equal(s.T(), "logstreamoutput", asyncResult.Value.Line)

		success <- true
	}()

	select {
	case <-fail:
		cancelFunc()
		s.T().Fail()
	case <-success:
		break
	}
}
