//go:build unit || !integration

package logstream_test

import (
	"github.com/stretchr/testify/require"

	"github.com/bacalhau-project/bacalhau/pkg/compute/store"
	dockermodels "github.com/bacalhau-project/bacalhau/pkg/executor/docker/models"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator"
	"github.com/bacalhau-project/bacalhau/pkg/test/mock"

	"github.com/bacalhau-project/bacalhau/pkg/docker"
	"github.com/bacalhau-project/bacalhau/pkg/executor"
	"github.com/bacalhau-project/bacalhau/pkg/system"
)

func (s *LogStreamTestSuite) TestStreamAddress() {
	docker.MustHaveDocker(s.T())
	node := s.stack.Nodes[0]

	es, err := dockermodels.NewDockerEngineBuilder("bash").
		WithEntrypoint("bash", "-c", "for i in {1..100}; do echo \"logstreamoutput\"; sleep 1; done").
		Build()
	s.Require().NoError(err)
	task := mock.TaskBuilder().
		Engine(es).
		BuildOrDie()
	job := mock.Job()
	job.Tasks[0] = task

	execution := mock.ExecutionForJob(job)
	execution.NodeID = node.ID
	execution.AllocateResources(task.Name, models.Resources{})

	err = node.RequesterNode.JobStore.CreateJob(s.ctx, *job)
	require.NoError(s.T(), err)

	exec, err := node.ComputeNode.Executors.Get(s.ctx, models.EngineDocker)
	require.NoError(s.T(), err)

	go func() {
		// Run the job.  We won't ever get a result because of the
		// entrypoint we chose, but we might get timed-out.
		exec.Run(
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
	}()

	// Wait for the docker container to be running so we know it'll be there when
	// the logstream requests it
	reader, err := waitForOutputStream(s.ctx, execution.ID, true, true, exec)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), reader)

	localExecutionState := store.NewLocalExecutionState(execution, "nodeID")
	node.ComputeNode.ExecutionStore.CreateExecution(s.ctx, *localExecutionState)

	execution.ComputeState.StateType = models.ExecutionStateBidAccepted
	err = node.RequesterNode.JobStore.CreateExecution(s.ctx, *execution)
	require.NoError(s.T(), err)

	ch, err := node.RequesterNode.Endpoint.ReadLogs(s.ctx, orchestrator.ReadLogsRequest{
		JobID:       job.ID,
		ExecutionID: execution.ID,
		Tail:        true,
		Follow:      true,
	})
	require.NoError(s.T(), err)

	asyncResult, ok := <-ch
	require.True(s.T(), ok)
	require.NoError(s.T(), asyncResult.Err)
	require.Equal(s.T(), models.ExecutionLogTypeSTDOUT, asyncResult.Value.Type)
	require.Equal(s.T(), "logstreamoutput\n", asyncResult.Value.Line)
}
