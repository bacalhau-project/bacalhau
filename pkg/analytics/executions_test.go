package analytics

import (
	"context"
	"testing"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/stretchr/testify/suite"
)

type ExecutionsTestSuite struct {
	suite.Suite
	ctx context.Context
}

func TestExecutionsSuite(t *testing.T) {
	suite.Run(t, new(ExecutionsTestSuite))
}

func (s *ExecutionsTestSuite) SetupTest() {
	s.ctx = context.Background()
}

func (s *ExecutionsTestSuite) createTestExecution() models.Execution {
	return models.Execution{
		ID:        "test-execution-id",
		JobID:     "test-job-id",
		EvalID:    "test-eval-id",
		NodeID:    "test-node-id",
		Namespace: "test-namespace",
		AllocatedResources: &models.AllocatedResources{
			Tasks: map[string]*models.Resources{
				"test-task": {
					CPU:    1.0,
					Memory: 1024,
					Disk:   2048,
					GPU:    1,
					GPUs: []models.GPU{
						{Name: "test-gpu", Vendor: "test-vendor"},
					},
				},
			},
		},
		DesiredState: models.State[models.ExecutionDesiredStateType]{
			StateType: models.ExecutionDesiredStateRunning,
			Details: map[string]string{
				models.DetailsKeyErrorCode: "test-error",
			},
		},
		ComputeState: models.State[models.ExecutionStateType]{
			StateType: models.ExecutionStateRunning,
			Message:   "test-message",
			Details: map[string]string{
				models.DetailsKeyErrorCode: "test-error",
			},
		},
		PublishedResult: &models.SpecConfig{
			Type: "test-publisher",
		},
		RunOutput: &models.RunCommandResult{
			StdoutTruncated: true,
			StderrTruncated: false,
			ExitCode:        0,
		},
		PreviousExecution: "prev-execution",
		NextExecution:     "next-execution",
		FollowupEvalID:    "followup-eval",
		Revision:          1,
		CreateTime:        time.Now().UnixNano(),
		ModifyTime:        time.Now().UnixNano(),
	}
}

func (s *ExecutionsTestSuite) TestCreatedExecutionEvent() {
	execution := s.createTestExecution()
	event := NewCreatedExecutionEvent(execution)
	s.Equal(CreatedExecutionEventType, event.Type)

	eventData, ok := event.Properties.(ExecutionEvent)
	s.True(ok, "Properties should be of type ExecutionEvent")

	s.Equal("test-job-id", eventData.JobID)
	s.Equal("test-execution-id", eventData.ExecutionID)
	s.Equal("test-eval-id", eventData.EvalID)
	s.Equal(hashString("test-node-id"), eventData.NodeNameHash)
	s.Equal(hashString("test-namespace"), eventData.NamespaceHash)
	s.Equal(models.ExecutionDesiredStateRunning.String(), eventData.DesiredState)
	s.Equal("test-error", eventData.DesiredStateErrorCode)
	s.Equal(models.ExecutionStateRunning.String(), eventData.ComputeState)
	s.Equal("test-error", eventData.ComputeStateErrorCode)
	s.Equal("test-publisher", eventData.PublishedResultType)
	s.True(eventData.RunResultStdoutTruncated)
	s.False(eventData.RunResultStderrTruncated)
	s.Equal(0, eventData.RunResultExitCode)
	s.Equal("prev-execution", eventData.PreviousExecution)
	s.Equal("next-execution", eventData.NextExecution)
	s.Equal("followup-eval", eventData.FollowupEvalID)
}

func (s *ExecutionsTestSuite) TestTerminalExecutionEvent() {
	execution := s.createTestExecution()
	event := NewTerminalExecutionEvent(execution)
	s.Equal(TerminalExecutionEventType, event.Type)

	eventData, ok := event.Properties.(ExecutionEvent)
	s.True(ok, "Properties should be of type ExecutionEvent")

	s.Equal("test-job-id", eventData.JobID)
	s.Equal("test-execution-id", eventData.ExecutionID)
	s.Equal("test-eval-id", eventData.EvalID)
	s.Equal(hashString("test-node-id"), eventData.NodeNameHash)
	s.Equal(hashString("test-namespace"), eventData.NamespaceHash)
	s.Equal(models.ExecutionDesiredStateRunning.String(), eventData.DesiredState)
	s.Equal("test-error", eventData.DesiredStateErrorCode)
	s.Equal(models.ExecutionStateRunning.String(), eventData.ComputeState)
	s.Equal("test-error", eventData.ComputeStateErrorCode)
	s.Equal("test-publisher", eventData.PublishedResultType)
	s.True(eventData.RunResultStdoutTruncated)
	s.False(eventData.RunResultStderrTruncated)
	s.Equal(0, eventData.RunResultExitCode)
	s.Equal("prev-execution", eventData.PreviousExecution)
	s.Equal("next-execution", eventData.NextExecution)
	s.Equal("followup-eval", eventData.FollowupEvalID)
}

func (s *ExecutionsTestSuite) TestComputeMessageEvent() {
	execution := models.Execution{
		ID:    "test-execution-id",
		JobID: "test-job-id",
		ComputeState: models.State[models.ExecutionStateType]{
			StateType: models.ExecutionStateRunning,
			Message:   "test-message",
			Details: map[string]string{
				models.DetailsKeyErrorCode: "test-error",
			},
		},
	}

	event := NewComputeMessageExecutionEvent(execution)
	s.Equal(ComputeMessageExecutionEventType, event.Type)

	eventData, ok := event.Properties.(ExecutionComputeMessage)
	s.True(ok, "Properties should be of type ExecutionComputeMessage")

	s.Equal("test-job-id", eventData.JobID)
	s.Equal("test-execution-id", eventData.ExecutionID)
	s.Equal("test-message", eventData.ComputeMessage)
	s.Equal("test-error", eventData.ComputeErrorCode)
}
