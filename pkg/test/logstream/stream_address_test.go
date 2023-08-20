//go:build unit || !integration

package logstream

import (
	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	dockermodels "github.com/bacalhau-project/bacalhau/pkg/executor/docker/models"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/test/mock"
	"github.com/stretchr/testify/require"

	"github.com/bacalhau-project/bacalhau/pkg/compute/logstream"
	"github.com/bacalhau-project/bacalhau/pkg/docker"
	"github.com/bacalhau-project/bacalhau/pkg/executor"
	"github.com/bacalhau-project/bacalhau/pkg/requester"
	"github.com/bacalhau-project/bacalhau/pkg/system"
)

func (s *LogStreamTestSuite) TestStreamAddress() {
	docker.MustHaveDocker(s.T())

	node := s.stack.Nodes[0]

	task := mock.TaskBuilder().
		Engine(dockermodels.NewDockerEngineBuilder("bash").
			WithEntrypoint("bash", "-c", "for i in {1..100}; do echo \"logstreamoutput\"; sleep 1; done").
			Build()).
		BuildOrDie()
	job := mock.Job()
	job.Tasks[0] = task

	execution := mock.ExecutionForJob(job)
	execution.NodeID = node.Host.ID().Pretty()
	execution.AllocateResources(task.Name, models.Resources{})

	err := node.RequesterNode.JobStore.CreateJob(s.ctx, *job)
	require.NoError(s.T(), err)

	exec, err := node.ComputeNode.Executors.Get(s.ctx, models.EngineDocker)
	require.NoError(s.T(), err)

	go func() {
		// Run the job.  We won't ever get a result because of the
		// entrypoint we chose, but we might get timed-out.
		_, err = exec.Run(
			s.ctx,
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
		s.NoError(err)
	}()

	// Wait for the docker container to be running so we know it'll be there when
	// the logstream requests it
	reader, err := waitForOutputStream(s.ctx, execution.ID, true, true, exec)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), reader)

	localExecutionState := store.NewLocalState(execution, "nodeID")
	node.ComputeNode.ExecutionStore.CreateExecution(s.ctx, *localExecutionState)

	execution.ComputeState.StateType = models.ExecutionStateBidAccepted
	err = node.RequesterNode.JobStore.CreateExecution(s.ctx, *execution)
	require.NoError(s.T(), err)

	logRequest := requester.ReadLogsRequest{
		JobID:       job.ID,
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
