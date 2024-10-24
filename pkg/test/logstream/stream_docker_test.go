//go:build unit || !integration

package logstream_test

import (
	"context"

	"github.com/stretchr/testify/require"

	dockermodels "github.com/bacalhau-project/bacalhau/pkg/executor/docker/models"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/test/mock"

	"github.com/bacalhau-project/bacalhau/pkg/docker"
	"github.com/bacalhau-project/bacalhau/pkg/executor"
	"github.com/bacalhau-project/bacalhau/pkg/system"
)

func (s *LogStreamTestSuite) TestDockerOutputStream() {
	docker.MustHaveDocker(s.T())

	node := s.stack.Nodes[0]

	ctx, cancelFunc := context.WithCancel(s.ctx)
	defer cancelFunc()

	success := make(chan bool, 1)
	fail := make(chan bool, 1)

	es, err := dockermodels.NewDockerEngineBuilder("busybox:latest").
		WithEntrypoint("sh", "-c", "for i in {1..100}; do echo \"logstreamoutput\"; sleep 1; done").
		Build()
	s.Require().NoError(err)
	task := mock.Task()
	task.Engine = es
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
		ch, err := waitForOutputStream(ctx, execution.ID, true, true, exec)
		require.NoError(s.T(), err)
		require.NotNil(s.T(), ch)

		asyncResult, ok := <-ch
		require.True(s.T(), ok)
		require.NoError(s.T(), asyncResult.Err)
		require.Equal(s.T(), models.ExecutionLogTypeSTDOUT, asyncResult.Value.Type)

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
