//go:build unit || !integration

package logstream

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/bacalhau-project/bacalhau/pkg/docker"
	"github.com/bacalhau-project/bacalhau/pkg/executor"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	testutils "github.com/bacalhau-project/bacalhau/pkg/test/utils"
)

func (s *LogStreamTestSuite) TestDockerOutputStream() {
	docker.MustHaveDocker(s.T())

	node := s.stack.Nodes[0]
	exec, err := node.ComputeNode.Executors.Get(s.ctx, model.EngineDocker)
	s.Require().NoError(err)

	ctx, cancelFunc := context.WithTimeout(s.ctx, time.Duration(10)*time.Second)
	defer cancelFunc()

	success := make(chan bool, 1)
	fail := make(chan bool, 1)

	job := testutils.MakeJob(
		model.EngineDocker,
		model.PublisherNoop,
		[]string{"bash", "-c", "for i in {1..100}; do echo \"logstreamoutput\"; sleep 1; done"})
	job.Metadata.ID = "logstreamtest-docker"

	node.RequesterNode.JobStore.CreateJob(ctx, *job)
	executionID := uuid.New().String()

	go func() {
		// Run the job.  We won't ever get a result because of the
		// entrypoint we chose, but we might get timed-out.
		var args *executor.Arguments
		if job.Spec.Engine == model.EngineDocker {
			args, err = executor.EncodeArguments(job.Spec.Docker)
			s.Require().NoError(err)
		}
		if job.Spec.Engine == model.EngineWasm {
			args, err = executor.EncodeArguments(job.Spec.Wasm)
			s.Require().NoError(err)
		}
		if job.Spec.Engine == model.EngineNoop {
			args = &executor.Arguments{Params: []byte{}}
		}
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
				EngineParams: args,
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
		s.Require().NoError(err)
		s.Require().NotNil(reader)

		dataframe, err := logger.NewDataFrameFromReader(reader)
		s.Require().NoError(err)

		s.Require().Contains(string(dataframe.Data), "logstreamoutput")

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
