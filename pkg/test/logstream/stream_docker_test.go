//go:build unit || !integration

package logstream

import (
	"context"
	"time"

	dockermodels "github.com/bacalhau-project/bacalhau/pkg/executor/docker/models"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/test/mock"
	"github.com/stretchr/testify/require"

	"github.com/bacalhau-project/bacalhau/pkg/docker"
	"github.com/bacalhau-project/bacalhau/pkg/executor"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/system"
)

func (s *LogStreamTestSuite) TestDockerOutputStream() {
	docker.MustHaveDocker(s.T())

	node := s.stack.Nodes[0]

	ctx, cancelFunc := context.WithTimeout(s.ctx, time.Duration(10)*time.Second)
	defer cancelFunc()

	success := make(chan bool, 1)
	fail := make(chan bool, 1)

	task := mock.TaskBuilder().
		Engine(dockermodels.NewDockerEngineBuilder("ubuntu:latest").
			WithEntrypoint("bash", "-c", "for i in {1..100}; do echo \"logstreamoutput\"; sleep 1; done").
			Build()).
		BuildOrDie()
	job := mock.Job()
	job.Tasks[0] = task

	execution := mock.ExecutionForJob(job)
	execution.AllocateResources(task.Name, models.Resources{})

	exec, err := node.ComputeNode.Executors.Get(s.ctx, models.EngineDocker)
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
		// TODO(forrest): [correctness] we need to wait a little for the container to become active.
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
