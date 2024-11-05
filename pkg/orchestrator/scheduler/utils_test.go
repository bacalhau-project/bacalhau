//go:build unit || !integration

package scheduler

import (
	"fmt"
	"testing"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/orchestrator"
	"github.com/bacalhau-project/bacalhau/pkg/test/mock"
)

type ScenarioBuilderOption func(*Scenario)

func WithJobType(jobType string) ScenarioBuilderOption {
	return func(b *Scenario) {
		b.job.Type = jobType
	}
}

func WithJobState(state models.JobStateType) ScenarioBuilderOption {
	return func(b *Scenario) {
		b.job.State.StateType = state
	}
}

func WithCount(count int) ScenarioBuilderOption {
	return func(b *Scenario) {
		b.job.Count = count
	}
}

func WithExecution(nodeID string, state models.ExecutionStateType) ScenarioBuilderOption {
	return func(b *Scenario) {
		execution := mock.ExecutionForJob(b.job)
		execution.NodeID = nodeID
		execution.ComputeState = models.NewExecutionState(state)
		b.executions = append(b.executions, *execution)
	}
}

func WithQueueTimeout(timeout time.Duration) ScenarioBuilderOption {
	return func(b *Scenario) {
		b.job.Task().Timeouts.QueueTimeout = int64(timeout.Seconds())
	}
}

func WithExecutionTimeout(timeout time.Duration) ScenarioBuilderOption {
	return func(b *Scenario) {
		b.job.Task().Timeouts.ExecutionTimeout = int64(timeout.Seconds())
	}
}

func WithTotalTimeout(timeout time.Duration) ScenarioBuilderOption {
	return func(b *Scenario) {
		b.job.Task().Timeouts.TotalTimeout = int64(timeout.Seconds())
	}
}

func WithCreateTime(t int64) ScenarioBuilderOption {
	return func(b *Scenario) {
		b.job.CreateTime = t
		if b.job.ModifyTime < t {
			b.job.ModifyTime = t
		}
	}
}

type Scenario struct {
	job        *models.Job
	executions []models.Execution
	evaluation *models.Evaluation
}

func NewScenario(opts ...ScenarioBuilderOption) *Scenario {
	job := mock.Job()
	job.Task().Timeouts = &models.TimeoutConfig{}

	builder := &Scenario{
		job:        job,
		executions: make([]models.Execution, 0),
		evaluation: models.NewEvaluation().WithJob(job),
	}
	for _, opt := range opts {
		opt(builder)
	}

	return builder
}

type ExpectedEvaluation struct {
	WaitUntil   time.Time
	TriggeredBy string
}

// Match verifies the observed evaluation matches fields from the source eval that initially triggered
// the scheduler, and the provided fields in the ExpectedEvaluation.
func (e ExpectedEvaluation) Match(t *testing.T, SourceEval, ObservedEval *models.Evaluation) bool {
	if SourceEval.JobID != ObservedEval.JobID {
		t.Logf("JobID: %s != %s", SourceEval.JobID, ObservedEval.JobID)
		return false
	}
	if SourceEval.Namespace != ObservedEval.Namespace {
		t.Logf("Namespace: %s != %s", SourceEval.Namespace, ObservedEval.Namespace)
		return false
	}
	if SourceEval.Priority != ObservedEval.Priority {
		t.Logf("Priority: %d != %d", SourceEval.Priority, ObservedEval.Priority)
		return false
	}
	if SourceEval.Type != ObservedEval.Type {
		t.Logf("Type: %s != %s", SourceEval.Type, ObservedEval.Type)
		return false
	}
	if models.EvalStatusPending != ObservedEval.Status {
		t.Logf("Status: %s != %s", models.EvalStatusPending, ObservedEval.Status)
		return false
	}
	if e.WaitUntil != ObservedEval.WaitUntil {
		t.Logf("WaitUntil: %s != %s", e.WaitUntil, ObservedEval.WaitUntil)
		return false
	}
	if e.TriggeredBy != ObservedEval.TriggeredBy {
		t.Logf("TriggerBy: %s != %s", e.TriggeredBy, ObservedEval.TriggeredBy)
		return false
	}
	return true
}

type PlanMatcher struct {
	t                         *testing.T
	JobState                  models.JobStateType
	Evaluation                *models.Evaluation
	NewExecutionsNodes        []string
	NewExecutionsDesiredState models.ExecutionDesiredStateType
	StoppedExecutions         []string
	ApprovedExecutions        []string
	NewEvaluations            []ExpectedEvaluation
}

type PlanMatcherParams struct {
	JobState                 models.JobStateType
	Evaluation               *models.Evaluation
	NewExecutionsNodes       []string
	NewExecutionDesiredState models.ExecutionDesiredStateType
	StoppedExecutions        []string
	ApprovedExecutions       []string
	ExpectedNewEvaluations   []ExpectedEvaluation
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
		NewEvaluations:            params.ExpectedNewEvaluations,
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
	for _, update := range plan.UpdatedExecutions {
		if update.DesiredState == models.ExecutionDesiredStateStopped {
			stoppedExecutions[update.Execution.ID] = struct{}{}
		}
		if update.DesiredState == models.ExecutionDesiredStateRunning {
			approvedExecutions[update.Execution.ID] = struct{}{}
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

	// check new evaluations match the expected evaluations
	if len(plan.NewEvaluations) != len(m.NewEvaluations) {
		m.t.Logf("NewEvaluations: %s != %s", plan.NewEvaluations, m.NewEvaluations)
		return false
	}
	for _, expectedEval := range m.NewEvaluations {
		found := false
		for _, planEval := range plan.NewEvaluations {
			if expectedEval.Match(m.t, m.Evaluation, planEval) {
				found = true
				break
			}
		}
		if !found {
			m.t.Logf("NewEvaluations: %s != %s", plan.NewEvaluations, m.NewEvaluations)
			return false
		}
	}
	return true
}

func (m PlanMatcher) String() string {
	return fmt.Sprintf("{JobState: %s, Evaluation: %s, NewExecutionsNodes: %s, StoppedExecutions: %s, ApprovedExecutions: %s}",
		m.JobState, m.Evaluation, m.NewExecutionsNodes, m.StoppedExecutions, m.ApprovedExecutions)
}

func fakeNodeInfo(t *testing.T, nodeID string) models.NodeInfo {
	return models.NodeInfo{
		NodeID: nodeID,
	}

}

func fakeNodeRank(t *testing.T, nodeID string) *orchestrator.NodeRank {
	return &orchestrator.NodeRank{
		NodeInfo: fakeNodeInfo(t, nodeID),
		Rank:     orchestrator.RankPreferred,
	}
}
