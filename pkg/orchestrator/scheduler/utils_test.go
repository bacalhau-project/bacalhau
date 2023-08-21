//go:build unit || !integration

package scheduler

import (
	"fmt"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/libp2p/go-libp2p/core/peer"
	"github.com/stretchr/testify/require"
)

type PlanMatcher struct {
	t                         *testing.T
	JobState                  models.JobStateType
	Evaluation                *models.Evaluation
	NewExecutionsNodes        []peer.ID
	NewExecutionsDesiredState models.ExecutionDesiredStateType
	StoppedExecutions         []string
	ApprovedExecutions        []string
}

type PlanMatcherParams struct {
	JobState                 models.JobStateType
	Evaluation               *models.Evaluation
	NewExecutionsNodes       []peer.ID
	NewExecutionDesiredState models.ExecutionDesiredStateType
	StoppedExecutions        []string
	ApprovedExecutions       []string
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
		desiredState, ok := newExecutionNodes[node.String()]
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

	return true
}

func (m PlanMatcher) String() string {
	return fmt.Sprintf("{JobState: %s, Evaluation: %s, NewExecutionsNodes: %s, StoppedExecutions: %s, ApprovedExecutions: %s}",
		m.JobState, m.Evaluation, m.NewExecutionsNodes, m.StoppedExecutions, m.ApprovedExecutions)
}

func mockNodeInfo(t *testing.T, nodeID string) *models.NodeInfo {
	id, err := peer.Decode(nodeID)
	require.NoError(t, err)
	return &models.NodeInfo{
		PeerInfo: peer.AddrInfo{
			ID: id,
		},
	}
}
