package scheduler

import (
	"fmt"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/libp2p/go-libp2p/core/peer"
)

type PlanMatcher struct {
	t                  *testing.T
	JobState           model.JobStateType
	Evaluation         *models.Evaluation
	NewExecutionsNodes []peer.ID
	StoppedExecutions  []model.ExecutionID
	ApprovedExecutions []model.ExecutionID
}

type PlanMatcherParams struct {
	JobState           model.JobStateType
	Evaluation         *models.Evaluation
	NewExecutionsNodes []peer.ID
	StoppedExecutions  []model.ExecutionID
	ApprovedExecutions []model.ExecutionID
}

// NewPlanMatcher returns a PlanMatcher with the given parameters.
func NewPlanMatcher(t *testing.T, params PlanMatcherParams) PlanMatcher {
	return PlanMatcher{
		t:                  t,
		JobState:           params.JobState,
		Evaluation:         params.Evaluation,
		NewExecutionsNodes: params.NewExecutionsNodes,
		StoppedExecutions:  params.StoppedExecutions,
		ApprovedExecutions: params.ApprovedExecutions,
	}
}

func (m PlanMatcher) Matches(x interface{}) bool {
	plan, ok := x.(*models.Plan)
	if !ok {
		return false
	}

	if plan.DesiredJobState != m.JobState {
		m.t.Logf("JobState: %s != %s", plan.DesiredJobState, m.JobState)
		return false
	}
	if plan.Eval != m.Evaluation || plan.EvalID != m.Evaluation.ID {
		m.t.Logf("Evaluation: %s != %s", plan.Eval, m.Evaluation)
		return false
	}

	// check new executions match the expected nodes
	newExecutionNodes := make(map[string]struct{})
	for _, execution := range plan.NewExecutions {
		newExecutionNodes[execution.NodeID] = struct{}{}
	}
	if len(newExecutionNodes) != len(m.NewExecutionsNodes) {
		m.t.Logf("NewExecutionsNodes: %s != %s", newExecutionNodes, m.NewExecutionsNodes)
		return false
	}
	for _, node := range m.NewExecutionsNodes {
		if _, ok := newExecutionNodes[node.String()]; !ok {
			m.t.Logf("NewExecutionsNodes: %s != %s", newExecutionNodes, m.NewExecutionsNodes)
			return false
		}
	}

	stoppedExecutions := make(map[model.ExecutionID]struct{})
	approvedExecutions := make(map[model.ExecutionID]struct{})
	for _, execution := range plan.UpdatedExecutions {
		if execution.DesiredState == model.ExecutionDesiredStateStopped {
			stoppedExecutions[execution.Execution.ID()] = struct{}{}
		}
		if execution.DesiredState == model.ExecutionDesiredStateRunning {
			approvedExecutions[execution.Execution.ID()] = struct{}{}
		}
	}

	// check stopped executions match the expected executions
	if len(stoppedExecutions) != len(m.StoppedExecutions) {
		m.t.Logf("StoppedExecutions: %s != %s", stoppedExecutions, m.StoppedExecutions)
		return false
	}
	for _, execution := range m.StoppedExecutions {
		if _, ok := stoppedExecutions[execution]; !ok {
			m.t.Logf("StoppedExecutions: %s != %s", stoppedExecutions, m.StoppedExecutions)
			return false
		}
	}

	// check approved executions match the expected executions
	if len(approvedExecutions) != len(m.ApprovedExecutions) {
		m.t.Logf("ApprovedExecutions: %s != %s", approvedExecutions, m.ApprovedExecutions)
		return false
	}
	for _, execution := range m.ApprovedExecutions {
		if _, ok := approvedExecutions[execution]; !ok {
			m.t.Logf("ApprovedExecutions: %s != %s", approvedExecutions, m.ApprovedExecutions)
			return false
		}
	}

	return true
}

func (m PlanMatcher) String() string {
	return fmt.Sprintf("{JobState: %s, Evaluation: %s, NewExecutionsNodes: %s, StoppedExecutions: %s, ApprovedExecutions: %s}",
		m.JobState, m.Evaluation, m.NewExecutionsNodes, m.StoppedExecutions, m.ApprovedExecutions)
}
