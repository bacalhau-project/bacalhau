package planner

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/compute"
	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/test/mock"
	"github.com/stretchr/testify/assert"
)

func mockCreateExecutions(plan *models.Plan) (*model.ExecutionState, *model.ExecutionState) {
	execution1 := mock.ExecutionState(plan.Job.ID())
	execution2 := mock.ExecutionState(plan.Job.ID())
	execution1.ComputeReference = "NewExec1"
	execution2.ComputeReference = "NewExec2"
	plan.NewExecutions = []*model.ExecutionState{execution1, execution2}
	return execution1, execution2
}

func mockUpdateExecutions(plan *models.Plan) (*models.PlanExecutionDesiredUpdate, *models.PlanExecutionDesiredUpdate) {
	execution1 := mock.ExecutionState(plan.Job.ID())
	execution2 := mock.ExecutionState(plan.Job.ID())
	execution1.ComputeReference = "UpdatedExec1"
	execution2.ComputeReference = "UpdatedExec2"
	update1 := &models.PlanExecutionDesiredUpdate{
		Execution:    execution1,
		DesiredState: model.ExecutionDesiredStateRunning,
		Comment:      "update 1",
	}
	update2 := &models.PlanExecutionDesiredUpdate{
		Execution:    execution2,
		DesiredState: model.ExecutionDesiredStateStopped,
		Comment:      "update 2",
	}
	plan.UpdatedExecutions[execution1.ID()] = update1
	plan.UpdatedExecutions[execution2.ID()] = update2
	return update1, update2
}

// UpdateExecutionMatcher is a matcher for the UpdateExecutionState method of the JobStore interface.
type UpdateExecutionMatcher struct {
	t               *testing.T
	execution       *model.ExecutionState
	newState        model.ExecutionStateType
	newDesiredState model.ExecutionDesiredState
	comment         string
	expectedState   model.ExecutionStateType
	expectedVersion int
}

type UpdateExecutionMatcherParams struct {
	NewState        model.ExecutionStateType
	NewDesiredState model.ExecutionDesiredState
	Comment         string
	ExpectedState   model.ExecutionStateType
	ExpectedVersion int
}

func NewUpdateExecutionMatcher(t *testing.T, execution *model.ExecutionState, params UpdateExecutionMatcherParams) *UpdateExecutionMatcher {
	return &UpdateExecutionMatcher{
		t:               t,
		execution:       execution,
		newState:        params.NewState,
		newDesiredState: params.NewDesiredState,
		comment:         params.Comment,
		expectedState:   params.ExpectedState,
		expectedVersion: params.ExpectedVersion,
	}
}

func NewUpdateExecutionMatcherFromPlanUpdate(t *testing.T, update *models.PlanExecutionDesiredUpdate) *UpdateExecutionMatcher {
	return NewUpdateExecutionMatcher(t, update.Execution, UpdateExecutionMatcherParams{
		NewDesiredState: update.DesiredState,
		Comment:         update.Comment,
		ExpectedVersion: update.Execution.Version,
	})
}

func (m *UpdateExecutionMatcher) Matches(x interface{}) bool {
	req, ok := x.(jobstore.UpdateExecutionRequest)
	if !ok {
		return false
	}

	// base expected request
	expectedRequest := jobstore.UpdateExecutionRequest{
		ExecutionID: m.execution.ID(),
		NewValues: model.ExecutionState{
			State:        m.newState,
			DesiredState: m.newDesiredState,
			Status:       m.comment,
		},
		Condition: jobstore.UpdateExecutionCondition{
			ExpectedVersion: m.expectedVersion,
		},
		Comment: m.comment,
	}

	// set expected state if present
	if !m.expectedState.IsUndefined() {
		expectedRequest.Condition.ExpectedStates = []model.ExecutionStateType{m.expectedState}
	}

	return reflect.DeepEqual(expectedRequest, req)
}

func (m *UpdateExecutionMatcher) String() string {
	return fmt.Sprintf("{Execution: %s}", m.execution)
}

// UpdateJobMatcher is a matcher for the UpdateJobState method of the JobStore interface.
type UpdateJobMatcher struct {
	t               *testing.T
	job             *model.Job
	newState        model.JobStateType
	comment         string
	expectedVersion int
}

type UpdateJobMatcherParams struct {
	NewState        model.JobStateType
	Comment         string
	ExpectedVersion int
}

func NewUpdateJobMatcher(t *testing.T, job *model.Job, params UpdateJobMatcherParams) *UpdateJobMatcher {
	return &UpdateJobMatcher{
		t:               t,
		job:             job,
		newState:        params.NewState,
		comment:         params.Comment,
		expectedVersion: params.ExpectedVersion,
	}
}

func NewUpdateJobMatcherFromPlanUpdate(t *testing.T, plan *models.Plan) *UpdateJobMatcher {
	return NewUpdateJobMatcher(t, plan.Job, UpdateJobMatcherParams{
		NewState:        plan.DesiredJobState,
		Comment:         plan.Comment,
		ExpectedVersion: plan.JobStateVersion,
	})
}

func (m *UpdateJobMatcher) Matches(x interface{}) bool {
	req, ok := x.(jobstore.UpdateJobStateRequest)
	if !ok {
		return false
	}

	// base expected request
	expectedRequest := jobstore.UpdateJobStateRequest{
		JobID:    m.job.ID(),
		NewState: m.newState,
		Comment:  m.comment,
		Condition: jobstore.UpdateJobCondition{
			ExpectedVersion: m.expectedVersion,
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
	execution *model.ExecutionState
	update    *models.PlanExecutionDesiredUpdate
}

func NewComputeRequestMatcher(t *testing.T, nodeID string, execution *model.ExecutionState) *ComputeRequestMatcher {
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
		executionID = req.ExecutionID
		desiredState := m.execution.DesiredState
		if m.update != nil {
			desiredState = m.update.DesiredState
		}
		if desiredState == model.ExecutionDesiredStatePending {
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

	return m.execution.ComputeReference == executionID &&
		m.nodeID == routingMetadata.SourcePeerID &&
		m.execution.NodeID == routingMetadata.TargetPeerID
}

func (m *ComputeRequestMatcher) String() string {
	return fmt.Sprintf("{Update Req: %+v}", m.update)
}
