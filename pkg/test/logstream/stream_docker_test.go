//go:build unit || !integration

package logstream

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"

	"github.com/bacalhau-project/bacalhau/pkg/docker"
	"github.com/bacalhau-project/bacalhau/pkg/executor"
	jobutils "github.com/bacalhau-project/bacalhau/pkg/job"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	testutils "github.com/bacalhau-project/bacalhau/pkg/test/utils"
)

func (s *LogStreamTestSuite) TestDockerOutputStream() {
	docker.MustHaveDocker(s.T())

	node := s.stack.Nodes[0]
	exec, err := node.ComputeNode.Executors.Get(s.ctx, model.EngineDocker)
	require.NoError(s.T(), err)

	ctx, cancelFunc := context.WithTimeout(s.ctx, time.Duration(10)*time.Second)
	defer cancelFunc()

	success := make(chan bool, 1)
	fail := make(chan bool, 1)

	job := testutils.MakeJobWithOpts(s.T(),
		jobutils.WithEngineSpec(
			model.NewDockerEngineBuilder("ubuntu:latest").
				WithEntrypoint("bash", "-c", "for i in {1..100}; do echo \"logstreamoutput\"; sleep 1; done").
				Build(),
		),
	)
	job.Metadata.ID = "logstreamtest-docker"

	require.NoError(s.T(), node.RequesterNode.JobStore.CreateJob(ctx, job))
	executionID := uuid.New().String()

	go func() {
		// Run the job.  We won't ever get a result because of the
		// entrypoint we chose, but we might get timed-out.
		engineBytes, err := job.Spec.EngineSpec.Serialize()
		require.NoError(s.T(), err)
		exec.Run(
			ctx,
			&executor.RunCommandRequest{
				JobID:        job.ID(),
				ExecutionID:  executionID,
				Resources:    job.Spec.Resources,
				Network:      job.Spec.Network,
				Outputs:      job.Spec.Outputs,
				Inputs:       nil,
				ResultsDir:   "/tmp",
				EngineParams: &executor.Arguments{Params: engineBytes},
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
		reader, err := waitForOutputStream(ctx, executionID, true, true, exec)
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
