package planner

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/test/mock"
	"github.com/stretchr/testify/assert"
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
		Comment:      "update 1",
	}
	update2 := &models.PlanExecutionDesiredUpdate{
		Execution:    execution2,
		DesiredState: models.ExecutionDesiredStateStopped,
		Comment:      "update 2",
	}
	plan.UpdatedExecutions[execution1.ID] = update1
	plan.UpdatedExecutions[execution2.ID] = update2
	return update1, update2
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
		DesiredStateComment: update.Comment,
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
		Comment:          plan.Comment,
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
		Comment:  m.comment,
		Condition: jobstore.UpdateJobCondition{
			ExpectedRevision: m.expectedRevision,
		},
	}
	return reflect.DeepEqual(expectedRequest, req)
}

func (m *UpdateJobMatcher) String() string {
	return fmt.Sprintf("{Job: %s}", m.job)
}

// ComputeRequestMatcher is a matcher for compute requests,
// including AskForBidRequest, BidAcceptedRequest, BidRejectedRequest and CancelExecutionRequest.
type ComputeRequestMatcher struct {
	t         *testing.T
	nodeID    string
	plan      *models.Plan
	execution *models.Execution
	update    *models.PlanExecutionDesiredUpdate
}

func NewComputeRequestMatcher(t *testing.T, nodeID string, execution *models.Execution) *ComputeRequestMatcher {
	return &ComputeRequestMatcher{
		t:         t,
		nodeID:    nodeID,
		execution: execution,
	}
}

func NewComputeRequestMatcherFromPlanUpdate(t *testing.T, nodeID string, update *models.PlanExecutionDesiredUpdate) *ComputeRequestMatcher {
	return &ComputeRequestMatcher{
		t:         t,
		nodeID:    nodeID,
		execution: update.Execution,
		update:    update,
	}
}

func (m *ComputeRequestMatcher) Matches(x interface{}) bool {
	var routingMetadata compute.RoutingMetadata
	var executionID string

	switch x.(type) {
	case compute.AskForBidRequest:
		req := x.(compute.AskForBidRequest)
		routingMetadata = req.RoutingMetadata
		executionID = req.Execution.ID
		desiredState := m.execution.DesiredState.StateType
		if m.update != nil {
			desiredState = m.update.DesiredState
		}
		if desiredState == models.ExecutionDesiredStatePending {
			if !req.WaitForApproval {
				return false
			}
		} else {
			if req.WaitForApproval {
				return false
			}
		}
	case compute.BidAcceptedRequest:
		req := x.(compute.BidAcceptedRequest)
		routingMetadata = req.RoutingMetadata
		executionID = req.ExecutionID
	case compute.BidRejectedRequest:
		req := x.(compute.BidRejectedRequest)
		routingMetadata = req.RoutingMetadata
		executionID = req.ExecutionID
	case compute.CancelExecutionRequest:
		req := x.(compute.CancelExecutionRequest)
		routingMetadata = req.RoutingMetadata
		executionID = req.ExecutionID
	default:
		return assert.Fail(m.t, fmt.Sprintf("unexpected type %T", x))
	}

	return m.execution.ID == executionID &&
		m.nodeID == routingMetadata.SourcePeerID &&
		m.execution.NodeID == routingMetadata.TargetPeerID
}

func (m *ComputeRequestMatcher) String() string {
	return fmt.Sprintf("{Update Req: %+v}", m.update)
}
