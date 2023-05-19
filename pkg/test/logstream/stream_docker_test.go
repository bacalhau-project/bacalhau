//go:build unit || !integration

package logstream

import (
	"context"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/docker"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	testutils "github.com/bacalhau-project/bacalhau/pkg/test/utils"
	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
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

	job := testutils.MakeJob(
		model.EngineDocker,
		model.VerifierNoop,
		model.PublisherNoop,
		[]string{"bash", "-c", "for i in {1..100}; do echo \"logstreamoutput\"; sleep 1; done"})
	job.Metadata.ID = "logstreamtest-docker"

	node.RequesterNode.JobStore.CreateJob(ctx, *job)
	executionID := uuid.New().String()

	go func() {
		// Run the job.  We won't ever get a result because of the
		// entrypoint we chose, but we might get timed-out.
		_, _ = exec.Run(ctx, executionID, *job, "/tmp")
		fail <- true
	}()

	go func() {
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
