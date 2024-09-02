package planner

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/test/mock"
)

func mockCreateExecutions(plan *models.Plan) (*models.Execution, *models.Execution) {
	execution1 := mock.ExecutionForJob(plan.Job)
	execution2 := mock.ExecutionForJob(plan.Job)
	execution1.ID = "NewExec1"
	execution2.ID = "NewExec2"
	plan.NewExecutions = []*models.Execution{execution1, execution2}
	return execution1, execution2
}

func mockUpdateExecutions(plan *models.Plan) (*models.PlanExecutionDesiredUpdate, *models.PlanExecutionDesiredUpdate) {
	execution1 := mock.ExecutionForJob(plan.Job)
	execution2 := mock.ExecutionForJob(plan.Job)
	execution1.ID = "UpdatedExec1"
	execution2.ID = "UpdatedExec2"
	update1 := &models.PlanExecutionDesiredUpdate{
		Execution:    execution1,
		DesiredState: models.ExecutionDesiredStateRunning,
		Event:        models.Event{Message: "update 1"},
	}
	update2 := &models.PlanExecutionDesiredUpdate{
		Execution:    execution2,
		DesiredState: models.ExecutionDesiredStateStopped,
		Event:        models.Event{Message: "update 2"},
	}
	plan.UpdatedExecutions[execution1.ID] = update1
	plan.UpdatedExecutions[execution2.ID] = update2
	return update1, update2
}

func mockCreateEvaluations(plan *models.Plan) (*models.Evaluation, *models.Evaluation) {
	evaluation1 := mock.EvalForJob(plan.Job)
	evaluation2 := mock.EvalForJob(plan.Job)
	evaluation1.ID = "NewEval1"
	evaluation2.ID = "NewEval2"
	plan.NewEvaluations = []*models.Evaluation{evaluation1, evaluation2}
	return evaluation1, evaluation2
}

// UpdateExecutionMatcher is a matcher for the UpdateExecutionState method of the JobStore interface.
type UpdateExecutionMatcher struct {
	t                   *testing.T
	execution           *models.Execution
	newState            models.ExecutionStateType
	newDesiredState     models.ExecutionDesiredStateType
	newStateComment     string
	desiredStateComment string
	expectedState       models.ExecutionStateType
	expectedRevision    uint64
}

type UpdateExecutionMatcherParams struct {
	NewState            models.ExecutionStateType
	NewDesiredState     models.ExecutionDesiredStateType
	NewStateComment     string
	DesiredStateComment string
	ExpectedState       models.ExecutionStateType
	ExpectedRevision    uint64
}

func NewUpdateExecutionMatcher(t *testing.T, execution *models.Execution, params UpdateExecutionMatcherParams) *UpdateExecutionMatcher {
	return &UpdateExecutionMatcher{
		t:                   t,
		execution:           execution,
		newState:            params.NewState,
		newDesiredState:     params.NewDesiredState,
		newStateComment:     params.NewStateComment,
		desiredStateComment: params.DesiredStateComment,
		expectedState:       params.ExpectedState,
		expectedRevision:    params.ExpectedRevision,
	}
}

func NewUpdateExecutionMatcherFromPlanUpdate(t *testing.T, update *models.PlanExecutionDesiredUpdate) *UpdateExecutionMatcher {
	return NewUpdateExecutionMatcher(t, update.Execution, UpdateExecutionMatcherParams{
		NewDesiredState:     update.DesiredState,
		DesiredStateComment: update.Event.Message,
		ExpectedRevision:    update.Execution.Revision,
	})
}

func (m *UpdateExecutionMatcher) Matches(x interface{}) bool {
	req, ok := x.(jobstore.UpdateExecutionRequest)
	if !ok {
		return false
	}

	// base expected request
	expectedRequest := jobstore.UpdateExecutionRequest{
		ExecutionID: m.execution.ID,
		NewValues: models.Execution{
			ComputeState: models.NewExecutionState(m.newState).WithMessage(m.newStateComment),
			DesiredState: models.NewExecutionDesiredState(m.newDesiredState).WithMessage(m.desiredStateComment),
		},
		Condition: jobstore.UpdateExecutionCondition{
			ExpectedRevision: m.expectedRevision,
		},
	}

	// set expected state if present
	if !m.expectedState.IsUndefined() {
		expectedRequest.Condition.ExpectedStates = []models.ExecutionStateType{m.expectedState}
	}

	return reflect.DeepEqual(expectedRequest, req)
}

func (m *UpdateExecutionMatcher) String() string {
	return fmt.Sprintf("{ExecutionForJob: %s, NewState: {%s %s}, DesiredState: {%s %s}",
		m.execution, m.newState, m.newStateComment, m.newDesiredState, m.newDesiredState)
}

// UpdateJobMatcher is a matcher for the UpdateJobState method of the JobStore interface.
type UpdateJobMatcher struct {
	t                *testing.T
	job              *models.Job
	newState         models.JobStateType
	comment          string
	expectedRevision uint64
}

type UpdateJobMatcherParams struct {
	NewState         models.JobStateType
	Comment          string
	ExpectedRevision uint64
}

func NewUpdateJobMatcher(t *testing.T, job *models.Job, params UpdateJobMatcherParams) *UpdateJobMatcher {
	return &UpdateJobMatcher{
		t:                t,
		job:              job,
		newState:         params.NewState,
		comment:          params.Comment,
		expectedRevision: params.ExpectedRevision,
	}
}

func NewUpdateJobMatcherFromPlanUpdate(t *testing.T, plan *models.Plan) *UpdateJobMatcher {
	return NewUpdateJobMatcher(t, plan.Job, UpdateJobMatcherParams{
		NewState:         plan.DesiredJobState,
		Comment:          plan.UpdateMessage,
		ExpectedRevision: plan.Job.Revision,
	})
}

func (m *UpdateJobMatcher) Matches(x interface{}) bool {
	req, ok := x.(jobstore.UpdateJobStateRequest)
	if !ok {
		return false
	}

	// base expected request
	expectedRequest := jobstore.UpdateJobStateRequest{
		JobID:    m.job.ID,
		NewState: m.newState,
		Message:  m.comment,
		Condition: jobstore.UpdateJobCondition{
			ExpectedRevision: m.expectedRevision,
		},
	}
	return reflect.DeepEqual(expectedRequest, req)
}

func (m *UpdateJobMatcher) String() string {
	return fmt.Sprintf("{Job: %s}", m.job)
}
