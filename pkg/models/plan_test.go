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

func TestRunPlanTestSuite(t *testing.T) {
	suite.Run(t, new(PlanTestSuite))
}
