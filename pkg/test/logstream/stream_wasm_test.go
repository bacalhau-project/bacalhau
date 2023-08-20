//go:build unit || !integration

package logstream

import (
	"context"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/executor"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
	"github.com/bacalhau-project/bacalhau/pkg/storage/inline"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/bacalhau-project/bacalhau/pkg/test/mock"
	"github.com/bacalhau-project/bacalhau/testdata/wasm/cat"
	"github.com/stretchr/testify/require"
	"github.com/vincent-petithory/dataurl"

	wasmmodels "github.com/bacalhau-project/bacalhau/pkg/executor/wasm/models"
	"github.com/bacalhau-project/bacalhau/pkg/model"
)

func (s *LogStreamTestSuite) TestWasmOutputStream() {
	node := s.stack.Nodes[0]

	ctx, cancelFunc := context.WithTimeout(s.ctx, time.Duration(10)*time.Second)
	defer cancelFunc()

	success := make(chan bool, 1)
	fail := make(chan bool, 1)

	task := mock.TaskBuilder().
		Engine(&models.SpecConfig{
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
			},
		}.ToMap()).
		BuildOrDie()
	job := mock.Job()
	job.Tasks[0] = task

	execution := mock.ExecutionForJob(job)
	execution.AllocateResources(task.Name, models.Resources{})

	err := node.RequesterNode.JobStore.CreateJob(s.ctx, *job)
	require.NoError(s.T(), err)

	exec, err := node.ComputeNode.Executors.Get(s.ctx, model.EngineWasm)
	require.NoError(s.T(), err)

	go func() {
		// Run the job.  We won't ever get a result because of the
		// entrypoint we chose, but we might get timed-out.
		exec.Run(
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
		fail <- true
	}()

	go func() {
		// TODO(walid): [correctness] following the recommendation in docker test to wait for module to run
		time.Sleep(time.Second * 3)
		reader, err := waitForOutputStream(ctx, execution.ID, true, true, exec)
		require.NoError(s.T(), err)
		require.NotNil(s.T(), reader)

		dataframe, err := logger.NewDataFrameFromReader(reader)
		require.NoError(s.T(), err)

		require.Contains(s.T(), string(dataframe.Data), "logstreamoutput")

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
