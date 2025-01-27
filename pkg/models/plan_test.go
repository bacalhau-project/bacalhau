//go:build unit || !integration

package models_test

import (
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/test/mock"
)

type PlanTestSuite struct {
	suite.Suite
	job  *models.Job
	eval *models.Evaluation
	plan *models.Plan
}

func (s *PlanTestSuite) SetupTest() {
	s.job = mock.Job()
	s.eval = mock.Eval()
	s.plan = models.NewPlan(s.eval, s.job)
}

func (s *PlanTestSuite) TestNewPlan() {
	s.Equal(s.eval.ID, s.plan.EvalID)
	s.Equal(s.eval.Priority, s.plan.Priority)
	s.Equal(s.eval, s.plan.Eval)
	s.Equal(s.job, s.plan.Job)
	s.Empty(s.plan.NewExecutions)
}

func (s *PlanTestSuite) TestMarkJobFailed() {
	s.plan.MarkJobFailed(models.Event{Message: "Test failure"})
	s.Equal(models.JobStateTypeFailed, s.plan.DesiredJobState)
	s.Equal("Test failure", s.plan.UpdateMessage)
	s.Equal(1, len(s.plan.JobEvents))
	s.Equal("Test failure", s.plan.JobEvents[0].Message)
}

func (s *PlanTestSuite) TestMarkJobRunningIfApplicable() {
	// Test when DesiredJobState is already defined
	s.plan = models.NewPlan(s.eval, s.job)
	s.plan.DesiredJobState = models.JobStateTypeCompleted
	s.plan.MarkJobRunningIfEligible(models.Event{Message: "job running"})
	s.Equal(models.JobStateTypeCompleted, s.plan.DesiredJobState, "Should not change DesiredJobState if already defined")
	s.Equal(0, len(s.plan.JobEvents), "Should not append JobEvents if DesiredJobState is already defined")

	// Test when JobStateType is not Pending
	s.job.State = models.NewJobState(models.JobStateTypeRunning)
	s.plan = models.NewPlan(s.eval, s.job)
	s.plan.MarkJobRunningIfEligible(models.Event{Message: "job running"})
	s.Equal(models.JobStateTypeUndefined, s.plan.DesiredJobState, "Should remain Undefined if Job is not in Pending state")
	s.Equal(0, len(s.plan.JobEvents), "Should not append JobEvents if Job is not in Pending state")

	// Test when there are no running executions
	s.job.State = models.NewJobState(models.JobStateTypePending)
	s.plan = models.NewPlan(s.eval, s.job)
	s.plan.MarkJobRunningIfEligible(models.Event{Message: "job running"})
	s.Equal(models.JobStateTypeUndefined, s.plan.DesiredJobState, "Should remain Undefined if no running executions")
	s.Equal(0, len(s.plan.JobEvents), "Should not append JobEvents if no running executions")

	// Test when conditions are met to set DesiredJobState to Running
	s.job.State = models.NewJobState(models.JobStateTypePending)
	s.plan = models.NewPlan(s.eval, s.job)
	s.plan.AppendExecution(&models.Execution{DesiredState: models.NewExecutionDesiredState(models.ExecutionDesiredStateRunning)}, models.Event{})
	s.plan.MarkJobRunningIfEligible(models.Event{Message: "job running"})
	s.Equal(models.JobStateTypeRunning, s.plan.DesiredJobState, "Should set to Running when all conditions are met")
	s.Equal(1, len(s.plan.JobEvents), "Should append JobEvents when DesiredJobState is set to Running")
	s.Equal("job running", s.plan.JobEvents[0].Message)
	// Test when conditions are met to set DesiredJobState to Running
	s.job.State = models.NewJobState(models.JobStateTypePending)
	s.plan = models.NewPlan(s.eval, s.job)
	s.plan.AppendApprovedExecution(mock.ExecutionForJob(s.plan.Job), models.Event{Message: "job approved"})
	s.plan.MarkJobRunningIfEligible(models.Event{Message: "job running"})
	s.Equal(models.JobStateTypeRunning, s.plan.DesiredJobState, "Should set to Running when all conditions are met")
	s.Equal(1, len(s.plan.JobEvents), "Should append JobEvents when DesiredJobState is set to Running")
	s.Equal("job running", s.plan.JobEvents[0].Message)
}

func (s *PlanTestSuite) TestAppendExecution() {
	exec := mock.ExecutionForJob(s.job)
	s.plan.AppendExecution(exec, models.Event{Message: "execution created"})
	s.Equal(1, len(s.plan.NewExecutions), "NewExecutions should have length 1 after appending")
	s.NotEmpty(s.plan.ExecutionEvents[exec.ID], "ExecutionEvents should have an entry for the new execution")
	s.Equal("execution created", s.plan.ExecutionEvents[exec.ID][0].Message, "Should append event to ExecutionEvents")
}

func (s *PlanTestSuite) TestMarkJobCompleted() {
	s.plan.MarkJobCompleted(models.Event{Message: "job completed"})
	s.Equal(models.JobStateTypeCompleted, s.plan.DesiredJobState, "Should set DesiredJobState to Completed")
	s.Equal(0, len(s.plan.NewExecutions), "NewExecutions should be empty after marking job as completed")
	s.Equal(1, len(s.plan.JobEvents), "Should append JobEvents when DesiredJobState is set to Completed")
	s.Equal("job completed", s.plan.JobEvents[0].Message)
}

func (s *PlanTestSuite) TestHasPendingWork() {
	// Test completely empty plan
	emptyPlan := models.NewPlan(s.eval, s.job)
	s.True(emptyPlan.HasPendingWork(), "New plan should be empty")

	// Test plan with new executions
	planWithNewExec := models.NewPlan(s.eval, s.job)
	planWithNewExec.AppendExecution(mock.ExecutionForJob(s.job), models.Event{})
	s.False(planWithNewExec.HasPendingWork(), "Plan with new executions should not be empty")

	// Test plan with updated executions
	planWithUpdatedExec := models.NewPlan(s.eval, s.job)
	planWithUpdatedExec.AppendStoppedExecution(
		mock.ExecutionForJob(s.job),
		models.Event{},
		models.ExecutionStateCancelled,
	)
	s.False(planWithUpdatedExec.HasPendingWork(), "Plan with updated executions should not be empty")

	// Test plan with new evaluations
	planWithNewEval := models.NewPlan(s.eval, s.job)
	planWithNewEval.AppendEvaluation(mock.Eval())
	s.False(planWithNewEval.HasPendingWork(), "Plan with new evaluations should not be empty")

	// Verify that events don't affect emptiness
	planWithEvents := models.NewPlan(s.eval, s.job)
	planWithEvents.AppendJobEvent(models.Event{Message: "test event"})
	planWithEvents.AppendExecutionEvent("exec-1", models.Event{Message: "test event"})
	s.True(planWithEvents.HasPendingWork(), "Plan with only events should be empty")

	// Verify that desired state doesn't affect emptiness
	planWithState := models.NewPlan(s.eval, s.job)
	planWithState.MarkJobCompleted(models.Event{})
	s.True(planWithState.HasPendingWork(), "Plan with only state changes should be empty")

	// Test plan with mixed content
	mixedPlan := models.NewPlan(s.eval, s.job)
	mixedPlan.AppendExecution(mock.ExecutionForJob(s.job), models.Event{})
	mixedPlan.AppendJobEvent(models.Event{})
	s.False(mixedPlan.HasPendingWork(), "Plan with executions should not be empty regardless of other content")
}

func TestRunPlanTestSuite(t *testing.T) {
	suite.Run(t, new(PlanTestSuite))
}
