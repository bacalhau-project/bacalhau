//go:build unit || !integration

package scheduler

import (
	"fmt"
	"testing"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/models"
)

type PlanMatcher struct {
	t                         *testing.T
	JobState                  models.JobStateType
	Evaluation                *models.Evaluation
	NewExecutionsNodes        []string
	NewExecutionsDesiredState models.ExecutionDesiredStateType
	StoppedExecutions         []string
	ApprovedExecutions        []string
	NewEvaluation             *models.Evaluation
}

type PlanMatcherParams struct {
	JobState                 models.JobStateType
	Evaluation               *models.Evaluation
	NewExecutionsNodes       []string
	NewExecutionDesiredState models.ExecutionDesiredStateType
	StoppedExecutions        []string
	ApprovedExecutions       []string
	NewEvaluation            *models.Evaluation
}

// NewPlanMatcher returns a PlanMatcher with the given parameters.
func NewPlanMatcher(t *testing.T, params PlanMatcherParams) PlanMatcher {
	return PlanMatcher{
		t:                         t,
		JobState:                  params.JobState,
		Evaluation:                params.Evaluation,
		NewExecutionsNodes:        params.NewExecutionsNodes,
		NewExecutionsDesiredState: params.NewExecutionDesiredState,
		StoppedExecutions:         params.StoppedExecutions,
		ApprovedExecutions:        params.ApprovedExecutions,
		NewEvaluation:             params.NewEvaluation,
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
	newExecutionNodes := make(map[string]models.ExecutionDesiredStateType)
	for _, execution := range plan.NewExecutions {
		newExecutionNodes[execution.NodeID] = execution.DesiredState.StateType
	}
	if len(newExecutionNodes) != len(m.NewExecutionsNodes) {
		m.t.Logf("NewExecutionsNodes: %v != %s", newExecutionNodes, m.NewExecutionsNodes)
		return false
	}
	for _, node := range m.NewExecutionsNodes {
		desiredState, ok := newExecutionNodes[node]
		if !ok {
			m.t.Logf("NewExecutionsNodes: %v != %s", newExecutionNodes, m.NewExecutionsNodes)
			return false
		}
		if desiredState != m.NewExecutionsDesiredState {
			m.t.Logf("NewExecutionsDesiredState: %v != %v", desiredState, m.NewExecutionsDesiredState)
			return false
		}
	}

	stoppedExecutions := make(map[string]struct{})
	approvedExecutions := make(map[string]struct{})
	for _, execution := range plan.UpdatedExecutions {
		if execution.DesiredState == models.ExecutionDesiredStateStopped {
			stoppedExecutions[execution.Execution.ID] = struct{}{}
		}
		if execution.DesiredState == models.ExecutionDesiredStateRunning {
			approvedExecutions[execution.Execution.ID] = struct{}{}
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

	if m.NewEvaluation != nil {
		if plan.NewEvaluation == nil {
			m.t.Logf("NewEvaluation: nil")
			return false
		}

		wanted := m.NewEvaluation
		got := plan.NewEvaluation

		if wanted.JobID != "" {
			if wanted.JobID != got.JobID {
				m.t.Logf("NewEvaluation.JobID: %s != %s", got.JobID, wanted.JobID)
				return false
			}
		}

		if wanted.TriggeredBy != "" {
			if wanted.TriggeredBy != got.TriggeredBy {
				m.t.Logf("NewEvaluation.TriggeredBy: %s != %s", got.TriggeredBy, wanted.TriggeredBy)
				return false
			}
		}

		if wanted.Type != "" {
			if wanted.Type != got.Type {
				m.t.Logf("NewEvaluation.Type: %s != %s", got.Type, wanted.Type)
				return false
			}
		}

		if !wanted.WaitUntil.IsZero() {
			// Sadly, the EvaluationBroker requires an absolute timestamp in the WaitUntil field. time.Now() will have advanced a little since the NewEvaluation was created, so we need to (sigh) allow some slack in the comparison here.
			difference := got.WaitUntil.Sub(wanted.WaitUntil)
			if difference < 0 || difference > time.Second {
				m.t.Logf("NewEvaluation.WaitUntil: %s != %s (difference %s is unacceptable)", got.WaitUntil, wanted.WaitUntil, difference)
				return false
			}
		}
	}

	return true
}

func (m PlanMatcher) String() string {
	return fmt.Sprintf("{JobState: %s, Evaluation: %s, NewExecutionsNodes: %s, StoppedExecutions: %s, ApprovedExecutions: %s}",
		m.JobState, m.Evaluation, m.NewExecutionsNodes, m.StoppedExecutions, m.ApprovedExecutions)
}

func mockNodeInfo(t *testing.T, nodeID string) *models.NodeInfo {
	return &models.NodeInfo{
		NodeID: nodeID,
	}
}
