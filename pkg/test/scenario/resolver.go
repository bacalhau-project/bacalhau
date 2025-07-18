package scenario

import (
	"context"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/jobstore"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/apimodels"
	"github.com/bacalhau-project/bacalhau/pkg/publicapi/client/v2"
	"github.com/bacalhau-project/bacalhau/pkg/system"
)

type StateResolver struct {
	loader          JobStateLoader
	maxWaitAttempts int
	waitDelay       time.Duration
}

type JobStateLoader interface {
	GetJob(ctx context.Context, id string) (*JobState, error)
}

type apiWrapper struct {
	api client.API
}

func (a *apiWrapper) GetJob(ctx context.Context, id string) (*JobState, error) {
	getResp, err := a.api.Jobs().Get(ctx, &apimodels.GetJobRequest{
		JobIDOrName: id,
		Include:     "executions",
	})
	if err != nil {
		return nil, fmt.Errorf("getting job from api: %w", err)
	}
	return &JobState{
		ID:         getResp.Job.ID,
		Executions: getResp.Executions.Items,
		State:      getResp.Job.State,
	}, nil
}

type storeWrapper struct {
	store jobstore.Store
}

func (s *storeWrapper) GetJob(ctx context.Context, id string) (*JobState, error) {
	job, err := s.store.GetJob(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("getting job from store: %w", err)
	}
	executions, err := s.store.GetExecutions(ctx, jobstore.GetExecutionsOptions{
		JobID:      id,
		IncludeJob: false,
	})
	if err != nil {
		return nil, fmt.Errorf("getting job executions from store: %w", err)
	}
	exe := make([]*models.Execution, 0, len(executions))
	for _, e := range executions {
		tmp := e
		exe = append(exe, &tmp)
	}
	return &JobState{
		ID:         job.ID,
		Executions: exe,
		State:      job.State,
	}, nil
}

func NewStateResolverFromStore(s jobstore.Store) *StateResolver {
	return &StateResolver{
		loader:          &storeWrapper{store: s},
		maxWaitAttempts: 1000,
		waitDelay:       time.Millisecond * 100,
	}
}

func NewStateResolverFromAPI(api client.API) *StateResolver {
	return &StateResolver{
		loader:          &apiWrapper{api: api},
		maxWaitAttempts: 1000,
		waitDelay:       time.Millisecond * 100,
	}
}

type JobState struct {
	ID         string
	Executions []*models.Execution
	State      models.State[models.JobStateType]
}

// GroupExecutionsByState groups the executions by state
func (s *JobState) GroupExecutionsByState() map[models.ExecutionStateType][]*models.Execution {
	result := make(map[models.ExecutionStateType][]*models.Execution)
	for _, execution := range s.Executions {
		result[execution.ComputeState.StateType] = append(result[execution.ComputeState.StateType], execution)
	}
	return result
}

type StateChecks func(s *JobState) (bool, error)

func (s *StateResolver) JobState(ctx context.Context, id string) (*JobState, error) {
	state, err := s.loader.GetJob(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get job (%s): %w", id, err)
	}

	return state, nil
}

func (s *StateResolver) Wait(ctx context.Context, id string, until ...StateChecks) error {
	waiter := &system.FunctionWaiter{
		Name:        "wait for job",
		MaxAttempts: s.maxWaitAttempts,
		Delay:       s.waitDelay,
		Handler: func() (bool, error) {
			state, err := s.JobState(ctx, id)
			if err != nil {
				return false, err
			}

			allOk := true
			for _, checkFunction := range until {
				stepOk, checkErr := checkFunction(state)
				if checkErr != nil {
					return false, checkErr
				}
				if !stepOk {
					allOk = false
				}
			}

			if allOk {
				return allOk, nil
			}

			// some of the check functions returned false
			// let's see if we can quit early because all expected states are
			// in terminal state
			allTerminal, err := WaitForTerminalStates()(state)
			if err != nil {
				return false, err
			}

			// If all the jobs are in terminal states, then nothing is going
			// to change if we keep polling, so we should exit early.
			if allTerminal {
				log.Ctx(ctx).Error().Msgf("all executions are in terminal state, but not all expected states are met: %+v", state)
				return false, fmt.Errorf("all jobs are in terminal states and conditions aren't met")
			}
			return false, nil
		},
	}

	return waiter.Wait(ctx)
}

func GetCompletedExecutionStates(jobState *JobState) []*models.Execution {
	return GetFilteredExecutionStates(jobState, models.ExecutionStateCompleted)
}

func GetFilteredExecutionStates(jobState *JobState, filterState models.ExecutionStateType) []*models.Execution {
	var ret []*models.Execution
	for _, executionState := range jobState.Executions {
		if executionState.ComputeState.StateType == filterState {
			ret = append(ret, executionState)
		}
	}
	return ret
}

// WaitForTerminalStates it is possible that a job is in a terminal state, but some executions are still running,
// such as when one node publishes the result before others.
// for that reason, we consider a job to be in a terminal state when:
// - all executions are in a terminal state
// - the job is in a terminal state to account for possible retries
// TODO validate this is comment is still valid.
func WaitForTerminalStates() StateChecks {
	return func(state *JobState) (bool, error) {
		for _, executionState := range state.Executions {
			if !executionState.ComputeState.StateType.IsTerminal() {
				return false, nil
			}
		}
		return state.State.StateType.IsTerminal(), nil
	}
}

// WaitForRunningState waits until the job is in a running state, or fails if in a terminal state
func WaitForRunningState() StateChecks {
	return func(jobState *JobState) (bool, error) {
		if jobState.State.StateType.IsTerminal() {
			return false, fmt.Errorf("job is in terminal state %s", jobState.State.StateType)
		}
		return jobState.State.StateType == models.JobStateTypeRunning, nil
	}
}

func WaitForSuccessfulCompletion() StateChecks {
	return func(jobState *JobState) (bool, error) {
		if jobState.State.StateType.IsTerminal() {
			if jobState.State.StateType != models.JobStateTypeCompleted {
				return false, fmt.Errorf("job did not complete successfully. "+
					"Completed with status: %s message: %s", jobState.State.StateType, jobState.State.Message)
			}
			return true, nil
		}
		return false, nil
	}
}

func WaitForUnsuccessfulCompletion() StateChecks {
	return func(jobState *JobState) (bool, error) {
		if jobState.State.StateType.IsTerminal() {
			if jobState.State.StateType != models.JobStateTypeFailed {
				return false, fmt.Errorf("job did not complete successfully")
			}
			return true, nil
		}
		return false, nil
	}
}

// WaitUntilSuccessful returns a set of job.CheckStatesFunctions that will wait
// until the job they are checking reaches the Completed state on the passed
// number of nodes. The checks will fail if any job errors.
func WaitUntilSuccessful(nodes int) []StateChecks {
	return []StateChecks{
		WaitExecutionsThrowErrors([]models.ExecutionStateType{
			models.ExecutionStateFailed,
		}),
		WaitForExecutionStates(map[models.ExecutionStateType]int{
			models.ExecutionStateCompleted: nodes,
		}),
	}
}

// error if there are any errors in any of the states
func WaitExecutionsThrowErrors(errorStates []models.ExecutionStateType) StateChecks {
	return func(jobState *JobState) (bool, error) {
		for _, execution := range jobState.Executions { //nolint:gocritic
			for _, errorState := range errorStates {
				if execution.ComputeState.StateType == errorState {
					e := log.Debug()
					if execution.RunOutput != nil {
						e = e.Str("stdout", execution.RunOutput.STDOUT).Str("stderr", execution.RunOutput.STDERR)
					}
					e.Msg("Job failed")
					return false, fmt.Errorf("job has error state %s on node %s (%s)",
						execution.ComputeState.StateType.String(), execution.NodeID, execution.ComputeState.Message)
				}
			}
		}
		return true, nil
	}
}

// wait for the given number of different states to occur
func WaitForExecutionStates(requiredStateCounts map[models.ExecutionStateType]int) StateChecks {
	return func(jobState *JobState) (bool, error) {
		discoveredStateCount := getExecutionStateTotals(jobState.Executions)
		log.Trace().Msgf("WaitForJobShouldHaveStates:\nrequired = %+v,\nactual = %+v\n", requiredStateCounts, discoveredStateCount)
		for requiredStateType, requiredStateCount := range requiredStateCounts {
			discoveredCount, ok := discoveredStateCount[requiredStateType]
			if !ok {
				discoveredCount = 0
			}
			if discoveredCount != requiredStateCount {
				return false, nil
			}
		}
		return true, nil
	}
}

func getExecutionStateTotals(executionStates []*models.Execution) map[models.ExecutionStateType]int {
	discoveredStateCount := map[models.ExecutionStateType]int{}
	for _, executionState := range executionStates { //nolint:gocritic
		discoveredStateCount[executionState.ComputeState.StateType]++
	}
	return discoveredStateCount
}
