//go:build unit || !integration

package logstream

import (
	"github.com/bacalhau-project/bacalhau/pkg/compute/logstream"
	"github.com/bacalhau-project/bacalhau/pkg/docker"
	"github.com/bacalhau-project/bacalhau/pkg/executor"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/requester"
	"github.com/bacalhau-project/bacalhau/pkg/system"
)

func (s *LogStreamTestSuite) TestStreamAddress() {
	docker.MustHaveDocker(s.T())

	node := s.stack.Nodes[0]

	job := newDockerJob("address-test", model.JobSpecDocker{
		Image:      "bash",
		Entrypoint: []string{"bash", "-c", "for i in {1..100}; do echo \"logstreamoutput\"; sleep 1; done"},
	})

	execution := newTestExecution("test-execution", job)

	err := node.RequesterNode.JobStore.CreateJob(s.ctx, job)
	s.Require().NoError(err)

	exec, err := node.ComputeNode.Executors.Get(s.ctx, model.EngineDocker)
	s.Require().NoError(err)

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
			s.ctx,
			&executor.RunCommandRequest{
				JobID:        job.ID(),
				ExecutionID:  execution.ID,
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
	}()

	// Wait for the docker container to be running so we know it'll be there when
	// the logstream requests it
	reader, err := waitForOutputStream(s.ctx, execution.ID, true, true, exec)
	s.Require().NoError(err)
	s.Require().NotNil(reader)

	node.ComputeNode.ExecutionStore.CreateExecution(s.ctx, execution)
	err = node.RequesterNode.JobStore.CreateExecution(s.ctx, model.ExecutionState{
		State:            model.ExecutionStateBidAccepted,
		JobID:            job.ID(),
		ComputeReference: execution.ID,
		NodeID:           node.Host.ID().Pretty(),
	})
	s.Require().NoError(err)

	logRequest := requester.ReadLogsRequest{
		JobID:       job.ID(),
		ExecutionID: execution.ID,
		WithHistory: true,
		Follow:      true}
	response, err := node.RequesterNode.Endpoint.ReadLogs(s.ctx, logRequest)
	s.Require().NoError(err)

	client, err := logstream.NewLogStreamClient(s.ctx, response.Address)
	s.Require().NoError(err)
	defer client.Close()

	client.Connect(s.ctx, execution.ID, true, true)

	frame, err := client.ReadDataFrame(s.ctx)
	s.Require().NoError(err)
	s.Require().NotNil(frame)

	s.Require().Equal(string(frame.Data), "logstreamoutput\n")
}
