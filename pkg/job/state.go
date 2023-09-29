package job

import (
	"context"
	"fmt"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/rs/zerolog/log"
)

type JobLoader func(ctx context.Context, id string) (model.Job, error)
type StateLoader func(ctx context.Context, id string) (model.JobState, error)

// a function that is given a map of nodeid -> job states
// and will throw an error if anything about that is wrong
type CheckStatesFunction func(model.JobState) (bool, error)

type StateResolver struct {
	jobLoader       JobLoader
	stateLoader     StateLoader
	maxWaitAttempts int
	waitDelay       time.Duration
}

func NewStateResolver(
	jobLoader JobLoader,
	stateLoader StateLoader,
) *StateResolver {
	return &StateResolver{
		jobLoader:       jobLoader,
		stateLoader:     stateLoader,
		maxWaitAttempts: 1000,
		waitDelay:       time.Millisecond * 100,
	}
}

func (resolver *StateResolver) GetJob(ctx context.Context, id string) (model.Job, error) {
	return resolver.jobLoader(ctx, id)
}

func (resolver *StateResolver) GetJobState(ctx context.Context, id string) (model.JobState, error) {
	return resolver.stateLoader(ctx, id)
}

func (resolver *StateResolver) SetWaitTime(maxWaitAttempts int, delay time.Duration) {
	resolver.maxWaitAttempts = maxWaitAttempts
	resolver.waitDelay = delay
}

func (resolver *StateResolver) GetExecutions(ctx context.Context, jobID string) ([]model.ExecutionState, error) {
	jobState, err := resolver.stateLoader(ctx, jobID)
	if err != nil {
		return []model.ExecutionState{}, err
	}
	return FlattenExecutionStates(jobState), nil
}

func (resolver *StateResolver) StateSummary(ctx context.Context, jobID string) (string, error) {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/job.StateResolver.StateSummary")
	defer span.End()

	jobState, err := resolver.stateLoader(ctx, jobID)
	if err != nil {
		return "", err
	}

	var currentJobState model.ExecutionStateType
	for _, executionState := range FlattenExecutionStates(jobState) { //nolint:gocritic
		if executionState.State > currentJobState {
			currentJobState = executionState.State
		}
	}

	return currentJobState.String(), nil
}

func (resolver *StateResolver) Wait(
	ctx context.Context,
	jobID string,
	checkJobStateFunctions ...CheckStatesFunction,
) error {
	return resolver.WaitWithOptions(ctx, WaitOptions{
		JobID: jobID,
	}, checkJobStateFunctions...)
}

type WaitOptions struct {
	// the job we are waiting for
	JobID string
	// in some cases we are actually waiting for an error state
	// this switch makes that OK (i.e. we don't try to return early)
	AllowAllTerminal bool
}

func (resolver *StateResolver) WaitWithOptions(
	ctx context.Context,
	options WaitOptions,
	checkJobStateFunctions ...CheckStatesFunction,
) error {
	waiter := &system.FunctionWaiter{
		Name:        "wait for job",
		MaxAttempts: resolver.maxWaitAttempts,
		Delay:       resolver.waitDelay,
		Handler: func() (bool, error) {
			jobState, err := resolver.stateLoader(ctx, options.JobID)
			if err != nil {
				return false, err
			}

			allOk := true
			for _, checkFunction := range checkJobStateFunctions {
				stepOk, checkErr := checkFunction(jobState)
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
			allTerminal, err := WaitForTerminalStates()(jobState)
			if err != nil {
				return false, err
			}

			// If all the jobs are in terminal states, then nothing is going
			// to change if we keep polling, so we should exit early.
			if allTerminal && !options.AllowAllTerminal {
				log.Ctx(ctx).Error().Msgf("all executions are in terminal state, but not all expected states are met: %+v", jobState)
				return false, fmt.Errorf("all jobs are in terminal states and conditions aren't met")
			}
			return false, nil
		},
	}

	return waiter.Wait(ctx)
}

// this is an auto wait where we auto calculate how many execution
// states we expect to see and we use that to pass to WaitForExecutionStates
func (resolver *StateResolver) WaitUntilComplete(ctx context.Context, jobID string) error {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/job.StateResolver.WaitUntilComplete")
	defer span.End()

	return resolver.Wait(
		ctx,
		jobID,
		WaitForSuccessfulCompletion(),
	)
}

func (resolver *StateResolver) GetResults(ctx context.Context, jobID string) ([]model.PublishedResult, error) {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/job.StateResolver.GetResults")
	defer span.End()

	results := []model.PublishedResult{}
	jobState, err := resolver.stateLoader(ctx, jobID)
	if err != nil {
		return results, err
	}

	for _, executionState := range GetCompletedExecutionStates(jobState) {
		results = append(results, model.PublishedResult{
			NodeID: executionState.NodeID,
			Data:   executionState.PublishedResult,
		})
	}

	return results, nil
}

func FlattenExecutionStates(jobState model.JobState) []model.ExecutionState {
	return jobState.Executions
}

func GetFilteredExecutionStates(jobState model.JobState, filterState model.ExecutionStateType) []model.ExecutionState {
	var ret []model.ExecutionState
	for _, executionState := range jobState.Executions {
		if executionState.State == filterState {
			ret = append(ret, executionState)
		}
	}
	return ret
}

func GetCompletedExecutionStates(jobState model.JobState) []model.ExecutionState {
	return GetFilteredExecutionStates(jobState, model.ExecutionStateCompleted)
}

func GetExecutionStateTotals(executionStates []model.ExecutionState) map[model.ExecutionStateType]int {
	discoveredStateCount := map[model.ExecutionStateType]int{}
	for _, executionState := range executionStates { //nolint:gocritic
		discoveredStateCount[executionState.State]++
	}
	return discoveredStateCount
}

// error if there are any errors in any of the states
func WaitExecutionsThrowErrors(errorStates []model.ExecutionStateType) CheckStatesFunction {
	return func(jobState model.JobState) (bool, error) {
		allExecutionStates := FlattenExecutionStates(jobState)
		for _, execution := range allExecutionStates { //nolint:gocritic
			for _, errorState := range errorStates {
				if execution.State == errorState {
					e := log.Debug()
					if execution.RunOutput != nil {
						e = e.Str("stdout", execution.RunOutput.STDOUT).Str("stderr", execution.RunOutput.STDERR)
					}
					e.Msg("Job failed")
					return false, fmt.Errorf("job has error state %s on node %s (%s)", execution.State.String(), execution.NodeID, execution.Status)
				}
			}
		}
		return true, nil
	}
}

// wait for the given number of different states to occur
func WaitForExecutionStates(requiredStateCounts map[model.ExecutionStateType]int) CheckStatesFunction {
	return func(jobState model.JobState) (bool, error) {
		executionStates := FlattenExecutionStates(jobState)
		discoveredStateCount := GetExecutionStateTotals(executionStates)
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

// WaitForTerminalStates it is possible that a job is in a terminal state, but some executions are still running,
// such as when one node publishes the result before others.
// for that reason, we consider a job to be in a terminal state when:
// - all executions are in a terminal state
// - the job is in a terminal state to account for possible retries
func WaitForTerminalStates() CheckStatesFunction {
	return func(jobState model.JobState) (bool, error) {
		executionStates := FlattenExecutionStates(jobState)
		for _, executionState := range executionStates {
			if !executionState.State.IsTerminal() {
				return false, nil
			}
		}
		return jobState.State.IsTerminal(), nil
	}
}

func WaitForSuccessfulCompletion() CheckStatesFunction {
	return func(jobState model.JobState) (bool, error) {
		if jobState.State.IsTerminal() {
			if jobState.State != model.JobStateCompleted {
				return false, fmt.Errorf("job did not complete successfully")
			}
			return true, nil
		}
		return false, nil
	}
}

func WaitForUnsuccessfulCompletion() CheckStatesFunction {
	return func(jobState model.JobState) (bool, error) {
		if jobState.State.IsTerminal() {
			if jobState.State != model.JobStateError {
				return false, fmt.Errorf("job did not complete successfully")
			}
			return true, nil
		}
		return false, nil
	}
}

// if there are > X states then error
func WaitDontExceedCount(count int) CheckStatesFunction {
	return func(jobState model.JobState) (bool, error) {
		executionStates := FlattenExecutionStates(jobState)
		if len(executionStates) > count {
			return false, fmt.Errorf("there are more states: %d than expected: %d", len(executionStates), count)
		}
		return true, nil
	}
}
