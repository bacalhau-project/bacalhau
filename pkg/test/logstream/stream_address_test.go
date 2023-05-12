//go:build unit || !integration

package logstream

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/compute/logstream"
	"github.com/bacalhau-project/bacalhau/pkg/docker"
	"github.com/bacalhau-project/bacalhau/pkg/executor"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/requester"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	noop_verifier "github.com/bacalhau-project/bacalhau/pkg/verifier/noop"
	"github.com/stretchr/testify/require"
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
	require.NoError(s.T(), err)

	exec, err := node.ComputeNode.Executors.Get(s.ctx, model.EngineDocker)
	require.NoError(s.T(), err)

	go func() {
		cm := system.NewCleanupManager()
		s.T().Cleanup(func() { cm.Cleanup(context.Background()) })

		result := s.T().TempDir()

		verifierMock, err := noop_verifier.NewNoopVerifierWithConfig(context.Background(), cm, noop_verifier.VerifierConfig{
			ExternalHooks: noop_verifier.VerifierExternalHooks{
				GetResultPath: func(ctx context.Context, executionID string, job model.Job) (string, error) {
					return result, nil
				},
			},
		})
		require.NoError(s.T(), err)

		ex, _ := node.ComputeNode.Executors.Get(s.ctx, model.EngineDocker)
		env, _ := executor.NewEnvironment(execution, ex.GetStorageProvider(s.ctx))
		env.Build(s.ctx, verifierMock)

		// Run the job.  We won't ever get a result because of the
		// entrypoint we chose, but we might get timed-out.
		_, _ = exec.Run(s.ctx, env)
	}()

	// Wait for the docker container to be running so we know it'll be there when
	// the logstream requests it
	reader, err := waitForOutputStream(s.ctx, execution.ID, true, true, exec)
	require.NoError(s.T(), err)
	require.NotNil(s.T(), reader)

	node.ComputeNode.ExecutionStore.CreateExecution(s.ctx, execution)
	err = node.RequesterNode.JobStore.CreateExecution(s.ctx, model.ExecutionState{
		State:             model.ExecutionStateBidAccepted,
		JobID:             job.ID(),
		ComputeReference:  execution.ID,
		AcceptedAskForBid: true,
		NodeID:            node.Host.ID().Pretty(),
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
