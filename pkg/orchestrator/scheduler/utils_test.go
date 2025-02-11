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

func WithPartitionedExecution(nodeID string, state models.ExecutionStateType, partitionIndex int) ScenarioBuilderOption {
	return func(b *Scenario) {
		execution := mock.ExecutionForJob(b.job)
		execution.NodeID = nodeID
		execution.ComputeState = models.NewExecutionState(state)
		execution.PartitionIndex = partitionIndex
		b.executions = append(b.executions, *execution)
	}
}

// WithDesiredState sets the desired state of the latest execution added to the scenario.
func WithDesiredState(state models.ExecutionDesiredStateType) ScenarioBuilderOption {
	return func(b *Scenario) {
		if len(b.executions) == 0 {
			panic("no executions to set desired state")
		}
		b.executions[len(b.executions)-1].DesiredState = models.NewExecutionDesiredState(state)
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

type ExecutionStateUpdate struct {
	ExecutionID  string
	DesiredState models.ExecutionDesiredStateType
	ComputeState models.ExecutionStateType
}

type PlanMatcher struct {
	t                 *testing.T
	JobState          models.JobStateType
	Evaluation        *models.Evaluation
	NewExecutions     []*models.Execution
	UpdatedExecutions []ExecutionStateUpdate
	NewEvaluations    []ExpectedEvaluation
}

type PlanMatcherParams struct {
	JobState               models.JobStateType
	Evaluation             *models.Evaluation
	NewExecutions          []*models.Execution
	UpdatedExecutions      []ExecutionStateUpdate
	ExpectedNewEvaluations []ExpectedEvaluation
}

// NewPlanMatcher returns a PlanMatcher with the given parameters.
func NewPlanMatcher(t *testing.T, params PlanMatcherParams) PlanMatcher {
	return PlanMatcher{
		t:                 t,
		JobState:          params.JobState,
		Evaluation:        params.Evaluation,
		NewExecutions:     params.NewExecutions,
		UpdatedExecutions: params.UpdatedExecutions,
		NewEvaluations:    params.ExpectedNewEvaluations,
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

	// If NewExecutions are specified, verify each one
	if len(plan.NewExecutions) != len(m.NewExecutions) {
		m.t.Logf("NewExecutions length: got %d, want %d", len(plan.NewExecutions), len(m.NewExecutions))
		return false
	}

	// group plan executions by node id
	planNewExecutions := make(map[string]*models.Execution)
	for _, execution := range plan.NewExecutions {
		planNewExecutions[execution.NodeID] = execution
	}

	for _, expectedExec := range m.NewExecutions {
		planExecution, ok := planNewExecutions[expectedExec.NodeID]
		if !ok {
			m.t.Logf("No new execution for node %s", expectedExec.NodeID)
			return false
		}

		// validate the desired state
		if planExecution.DesiredState.StateType != expectedExec.DesiredState.StateType {
			m.t.Logf("DesiredState: %s != %s for node %s", planExecution.DesiredState.StateType, expectedExec.DesiredState.StateType, expectedExec.NodeID)
			return false
		}

		// validate the partition index
		if planExecution.PartitionIndex != expectedExec.PartitionIndex {
			m.t.Logf("PartitionIndex: %d != %d for node %s", planExecution.PartitionIndex, expectedExec.PartitionIndex, expectedExec.NodeID)
			return false
		}
	}

	// Check updated executions
	if len(plan.UpdatedExecutions) != len(m.UpdatedExecutions) {
		m.t.Logf("UpdatedExecutions length: got %d, want %d",
			len(plan.UpdatedExecutions), len(m.UpdatedExecutions))
		return false
	}

	// Build map of expected updates for easier lookup
	expectedUpdates := make(map[string]ExecutionStateUpdate)
	for _, update := range m.UpdatedExecutions {
		expectedUpdates[update.ExecutionID] = update
	}

	// Verify each plan update matches expectations
	for _, update := range plan.UpdatedExecutions {
		execID := update.Execution.ID
		expected, ok := expectedUpdates[execID]
		if !ok {
			m.t.Logf("Unexpected execution update for %s", execID)
			return false
		}

		if update.DesiredState != expected.DesiredState {
			m.t.Logf("Execution %s DesiredState: got %s, want %s",
				execID, update.DesiredState, expected.DesiredState)
			return false
		}

		if update.ComputeState != expected.ComputeState {
			m.t.Logf("Execution %s ComputeState: got %s, want %s",
				execID, update.ComputeState, expected.ComputeState)
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
	return fmt.Sprintf("{JobState: %s, Evaluation: %s, NewExecutions: %s, UpdatedExecutions: %s}",
		m.JobState, m.Evaluation, m.NewExecutions, m.UpdatedExecutions)
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
