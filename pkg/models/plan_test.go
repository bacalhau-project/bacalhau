//go:build unit || !integration

package models_test

import (
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/test/mock"
	"github.com/stretchr/testify/suite"
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
	s.plan.MarkJobFailed("Test failure")
	s.Equal(models.JobStateTypeFailed, s.plan.DesiredJobState)
	s.Equal("Test failure", s.plan.Comment)
}

func (s *PlanTestSuite) TestMarkJobRunningIfApplicable() {
	// Test when DesiredJobState is already defined
	s.plan = models.NewPlan(s.eval, s.job)
	s.plan.DesiredJobState = models.JobStateTypeCompleted
	s.plan.MarkJobRunningIfEligible()
	s.Equal(models.JobStateTypeCompleted, s.plan.DesiredJobState, "Should not change DesiredJobState if already defined")

	// Test when JobStateType is not Pending
	s.job.State = models.NewJobState(models.JobStateTypeRunning)
	s.plan = models.NewPlan(s.eval, s.job)
	s.plan.MarkJobRunningIfEligible()
	s.Equal(models.JobStateTypeUndefined, s.plan.DesiredJobState, "Should remain Undefined if Job is not in Pending state")

	// Test when there are no running executions
	s.job.State = models.NewJobState(models.JobStateTypePending)
	s.plan = models.NewPlan(s.eval, s.job)
	s.plan.MarkJobRunningIfEligible()
	s.Equal(models.JobStateTypeUndefined, s.plan.DesiredJobState, "Should remain Undefined if no running executions")

	// Test when conditions are met to set DesiredJobState to Running
	s.job.State = models.NewJobState(models.JobStateTypePending)
	s.plan = models.NewPlan(s.eval, s.job)
	s.plan.AppendExecution(&models.Execution{DesiredState: models.NewExecutionDesiredState(models.ExecutionDesiredStateRunning)})
	s.plan.MarkJobRunningIfEligible()
	s.Equal(models.JobStateTypeRunning, s.plan.DesiredJobState, "Should set to Running when all conditions are met")

	// Test when conditions are met to set DesiredJobState to Running
	s.job.State = models.NewJobState(models.JobStateTypePending)
	s.plan = models.NewPlan(s.eval, s.job)
	s.plan.AppendApprovedExecution(mock.ExecutionForJob(s.plan.Job))
	s.plan.MarkJobRunningIfEligible()
	s.Equal(models.JobStateTypeRunning, s.plan.DesiredJobState, "Should set to Running when all conditions are met")
}

func (s *PlanTestSuite) TestAppendExecution() {
	s.plan.AppendExecution(mock.ExecutionForJob(s.job))
	s.Equal(1, len(s.plan.NewExecutions), "NewExecutions should have length 1 after appending")
}

func (s *PlanTestSuite) TestMarkJobCompleted() {
	s.plan.MarkJobCompleted()
	s.Equal(models.JobStateTypeCompleted, s.plan.DesiredJobState, "Should set DesiredJobState to Completed")
	s.Equal(0, len(s.plan.NewExecutions), "NewExecutions should be empty after marking job as completed")
}

func TestRunPlanTestSuite(t *testing.T) {
	suite.Run(t, new(PlanTestSuite))
}
