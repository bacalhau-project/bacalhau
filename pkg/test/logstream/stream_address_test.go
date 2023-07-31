//go:build unit || !integration

package logstream

import (
	"github.com/stretchr/testify/require"

	"github.com/bacalhau-project/bacalhau/pkg/compute/logstream"
	"github.com/bacalhau-project/bacalhau/pkg/docker"
	"github.com/bacalhau-project/bacalhau/pkg/executor"
	jobutils "github.com/bacalhau-project/bacalhau/pkg/job"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/requester"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	testutils "github.com/bacalhau-project/bacalhau/pkg/test/utils"
)

func (s *LogStreamTestSuite) TestStreamAddress() {
	docker.MustHaveDocker(s.T())

	node := s.stack.Nodes[0]

	job := testutils.MakeJobWithOpts(s.T(),
		jobutils.WithEngineSpec(
			model.NewDockerEngineBuilder("bash").
				WithEntrypoint("bash", "-c", "for i in {1..100}; do echo \"logstreamoutput\"; sleep 1; done").
				Build(),
		),
	)
	job.Metadata = model.Metadata{
		ID: "address-test",
	}

	execution := newTestExecution("test-execution", job)

	err := node.RequesterNode.JobStore.CreateJob(s.ctx, job)
	require.NoError(s.T(), err)

	exec, err := node.ComputeNode.Executors.Get(s.ctx, model.EngineDocker)
	require.NoError(s.T(), err)

	go func() {
		// Run the job.  We won't ever get a result because of the
		// entrypoint we chose, but we might get timed-out.
		engineArgs, err := job.Spec.EngineSpec.Serialize()
		require.NoError(s.T(), err)
		args := &executor.Arguments{Params: engineArgs}

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
	require.NoError(s.T(), err)
	require.NotNil(s.T(), reader)

	node.ComputeNode.ExecutionStore.CreateExecution(s.ctx, execution)
	err = node.RequesterNode.JobStore.CreateExecution(s.ctx, model.ExecutionState{
		State:            model.ExecutionStateBidAccepted,
		JobID:            job.ID(),
		ComputeReference: execution.ID,
		NodeID:           node.Host.ID().Pretty(),
	})
	require.NoError(s.T(), err)

	logRequest := requester.ReadLogsRequest{
		JobID:       job.ID(),
		ExecutionID: execution.ID,
		WithHistory: true,
		Follow:      true}
	response, err := node.RequesterNode.Endpoint.ReadLogs(s.ctx, logRequest)
	require.NoError(s.T(), err)

	client, err := logstream.NewLogStreamClient(s.ctx, response.Address)
	require.NoError(s.T(), err)
	defer client.Close()

	client.Connect(s.ctx, execution.ID, true, true)

	frame, err := client.ReadDataFrame(s.ctx)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), frame)

	require.Equal(s.T(), string(frame.Data), "logstreamoutput\n")
}
