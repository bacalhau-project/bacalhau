//go:build integration || !unit

package devstack

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/devstack"
	"github.com/bacalhau-project/bacalhau/pkg/docker"
	dockmodels "github.com/bacalhau-project/bacalhau/pkg/executor/docker/models"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/node"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	clientv2 "github.com/bacalhau-project/bacalhau/pkg/publicapi/client/v2"
	"github.com/bacalhau-project/bacalhau/pkg/test/scenario"
	"github.com/bacalhau-project/bacalhau/pkg/test/teststack"
	"github.com/bacalhau-project/bacalhau/pkg/util/idgen"

	"github.com/bacalhau-project/bacalhau/pkg/logger"
)

type StopSuite struct {
	suite.Suite
	requester     *node.Node
	compute       *node.Node
	client        clientv2.API
	stateResolver *scenario.StateResolver
}

func TestStopSuite(t *testing.T) {
	suite.Run(t, new(StopSuite))
}

func (s *StopSuite) SetupSuite() {
	logger.ConfigureTestLogging(s.T())
	docker.MustHaveDocker(s.T())
	ctx := context.Background()
	stack := teststack.Setup(ctx, s.T(),
		devstack.WithNumberOfHybridNodes(1),
	)
	s.requester = stack.Nodes[0]
	s.compute = stack.Nodes[0] // hybrid node
	s.client = clientv2.New(fmt.Sprintf("http://%s:%d", s.requester.APIServer.Address, s.requester.APIServer.Port))
	s.stateResolver = scenario.NewStateResolverFromStore(s.requester.RequesterNode.JobStore)
}

func (s *StopSuite) TearDownSuite() {
	if s.requester != nil {
		s.requester.CleanupManager.Cleanup(context.Background())
	}
}

func (s *StopSuite) TestStop_HappyPath() {
	ctx := context.Background()
	jobID, err := s.submitJob(10)
	s.Require().NoError(err)

	s.Require().NoError(s.stateResolver.Wait(ctx, jobID, scenario.WaitForRunningState()))

	evalID, err := s.stopJob(jobID, "test stop")
	s.Require().NoError(err)
	s.Require().NotEmpty(evalID)

	s.verifyJobState(jobID, models.JobStateTypeStopped, "test stop")
}

func (s *StopSuite) TestStop_ShortID() {
	ctx := context.Background()
	jobID, err := s.submitJob(10)
	s.Require().NoError(err)

	s.Require().NoError(s.stateResolver.Wait(ctx, jobID, scenario.WaitForRunningState()))

	evalID, err := s.stopJob(idgen.ShortUUID(jobID), "test stop")
	s.Require().NoError(err)
	s.Require().NotEmpty(evalID)

	s.verifyJobState(jobID, models.JobStateTypeStopped, "test stop")
}

func (s *StopSuite) TestStop_AlreadyCompleted() {
	ctx := context.Background()
	jobID, err := s.submitJob(0) // Short sleep so job completes quickly
	s.Require().NoError(err)

	s.Require().NoError(s.stateResolver.Wait(ctx, jobID, scenario.WaitForSuccessfulCompletion()))

	_, err = s.stopJob(jobID, "test stop")
	s.Require().Error(err)

	s.verifyJobState(jobID, models.JobStateTypeCompleted, "")
}

func (s *StopSuite) TestStop_AlreadyStopped() {
	ctx := context.Background()
	jobID, err := s.submitJob(10)
	s.Require().NoError(err)

	s.Require().NoError(s.stateResolver.Wait(ctx, jobID, scenario.WaitForRunningState()))

	evalID, err := s.stopJob(jobID, "first stop")
	s.Require().NoError(err)
	s.Require().NotEmpty(evalID)

	_, err = s.stopJob(jobID, "second stop")
	s.Require().NoError(err)

	s.verifyJobState(jobID, models.JobStateTypeStopped, "first stop")
}

func (s *StopSuite) TestStop_NotFound() {
	_, err := s.stopJob("j-nonexistent", "test stop")
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "not found")
}

func (s *StopSuite) TestStop_MultipleMatches() {
	ctx := context.Background()

	// Submit two jobs that will have a common prefix (usually just 'j-')
	jobID1, err := s.submitJob(10)
	s.Require().NoError(err)
	s.Require().NoError(s.stateResolver.Wait(ctx, jobID1, scenario.WaitForRunningState()))

	jobID2, err := s.submitJob(10)
	s.Require().NoError(err)
	s.Require().NoError(s.stateResolver.Wait(ctx, jobID2, scenario.WaitForRunningState()))

	// Extract common prefix (should be at least 'j-')
	commonPrefix := jobID1[:2]
	s.Require().True(strings.HasPrefix(jobID2, commonPrefix), "Jobs should share a common prefix")

	// Try to stop using the common prefix
	_, err = s.stopJob(commonPrefix, "test stop")
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "multiple jobs")

	// Verify neither job was stopped
	s.verifyJobState(jobID1, models.JobStateTypeRunning, "")
	s.verifyJobState(jobID2, models.JobStateTypeRunning, "")
}

// Helper function to create and submit a job
func (s *StopSuite) submitJob(sleepTime int) (string, error) {
	ctx := context.Background()
	j := &models.Job{
		Type:  models.JobTypeBatch,
		Count: 1,
		Tasks: []*models.Task{
			{
				Name: "main",
				Engine: dockmodels.NewDockerEngineBuilder("busybox:1.37.0").
					WithEntrypoint("sh", "-c", fmt.Sprintf("sleep %d", sleepTime)).
					MustBuild(),
			},
		},
	}
	submitResp, err := s.client.Jobs().Put(ctx, &apimodels.PutJobRequest{
		Job: j,
	})
	if err != nil {
		return "", err
	}
	return submitResp.JobID, nil
}

// Helper function to verify job state
func (s *StopSuite) verifyJobState(jobID string, expectedState models.JobStateType, expectedMessage string) {
	ctx := context.Background()
	getResp, err := s.client.Jobs().Get(ctx, &apimodels.GetJobRequest{
		JobID:   jobID,
		Include: "executions",
	})
	s.Require().NoError(err)
	s.Require().Equal(expectedState, getResp.Job.State.StateType)
	if expectedMessage != "" {
		s.Require().Equal(expectedMessage, getResp.Job.State.Message)
	}

	if expectedState == models.JobStateTypeStopped {
		// verify compute node state that the job stop request has been propagated and processed
		// get execution id from the response
		s.Require().NotNil(getResp.Executions)
		s.Require().NotEmpty(getResp.Executions.Items)
		executionID := getResp.Executions.Items[0].ID

		s.Eventually(func() bool {
			execution, err := s.compute.ComputeNode.ExecutionStore.GetExecution(ctx, executionID)
			if err != nil {
				return false
			}
			return execution.ComputeState.StateType == models.ExecutionStateCancelled
		}, 5*time.Second, 50*time.Millisecond)
	}
}

// Helper function to stop a job and verify the response
func (s *StopSuite) stopJob(jobID, reason string) (string, error) {
	ctx := context.Background()
	stopResp, err := s.client.Jobs().Stop(ctx, &apimodels.StopJobRequest{
		JobID:  jobID,
		Reason: reason,
	})
	if err != nil {
		return "", err
	}
	return stopResp.EvaluationID, nil
}
