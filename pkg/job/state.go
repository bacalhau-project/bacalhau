package job

import (
	"context"
	"fmt"
	"math"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/system"
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

func (resolver *StateResolver) GetShards(ctx context.Context, jobID string) ([]model.ExecutionState, error) {
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
	for _, shardState := range FlattenExecutionStates(jobState) { //nolint:gocritic
		if shardState.State > currentJobState {
			currentJobState = shardState.State
		}
	}

	return currentJobState.String(), nil
}

func (resolver *StateResolver) VerifiedSummary(ctx context.Context, jobID string) (string, error) {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/job.StateResolver.VerifiedSummary")
	defer span.End()

	j, err := resolver.jobLoader(ctx, jobID)
	if err != nil {
		return "", err
	}

	if j.Spec.Verifier == model.VerifierNoop {
		return "", nil
	}

	jobState, err := resolver.stateLoader(ctx, jobID)
	if err != nil {
		return "", err
	}
	totalShards := GetJobTotalExecutionCount(j)
	verifiedShardCount := CountVerifiedShardStates(jobState)

	return fmt.Sprintf("%d/%d", verifiedShardCount, totalShards), nil
}

func (resolver *StateResolver) ResultSummary(ctx context.Context, jobID string) (string, error) {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/job.StateResolver.ResultSummary")
	defer span.End()

	job, err := resolver.jobLoader(ctx, jobID)
	if err != nil {
		return "", err
	}
	if GetJobTotalShards(job) > 1 {
		return "", nil
	}
	jobState, err := resolver.stateLoader(ctx, jobID)
	if err != nil {
		return "", err
	}
	completedShards := GetCompletedVerifiedShardStates(jobState)
	if len(completedShards) == 0 {
		return "", nil
	}
	return fmt.Sprintf("/ipfs/%s", completedShards[0].PublishedResult.CID), nil
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

// this is an auto wait where we auto calculate how many shard
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

	// group the shard states by shard index
	for _, shardState := range GetCompletedVerifiedShardStates(jobState) {
		results = append(results, model.PublishedResult{
			NodeID:     shardState.NodeID,
			ShardIndex: shardState.ShardIndex,
			Data:       shardState.PublishedResult,
		})
	}

	return results, nil
}

type ShardStateChecker func(
	shardStates []model.ExecutionState,
	concurrency int,
) (bool, error)

// iterate each shard and pass off []model.ExecutionState to the given function
// every shard must return true for this function to return true
// this is useful for example to say "do we have enough to begin verification"
func (resolver *StateResolver) CheckShardStates(
	ctx context.Context,
	shard model.JobShard,
	shardStateChecker ShardStateChecker,
) (bool, error) {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/job.StateResolver.CheckShardStates")
	defer span.End()

	jobState, err := resolver.stateLoader(ctx, shard.Job.Metadata.ID)
	if err != nil {
		return false, err
	}

	concurrency := int(math.Max(float64(shard.Job.Spec.Deal.Concurrency), 1))
	shardStates := GetStatesForShardIndex(jobState, shard.Index)
	if len(shardStates) == 0 {
		return false, fmt.Errorf("job (%s) has no shard state for shard index %d", shard.Job.Metadata.ID, shard.Index)
	}

	shardCheckResult, err := shardStateChecker(shardStates, concurrency)
	if err != nil {
		return false, err
	}
	if !shardCheckResult {
		return false, nil
	}
	return true, nil
}

func FlattenExecutionStates(jobState model.JobState) []model.ExecutionState {
	var ret []model.ExecutionState
	for _, shardState := range jobState.Shards {
		ret = append(ret, shardState.Executions...)
	}
	return ret
}

func GetStatesForShardIndex(jobState model.JobState, shardIndex int) []model.ExecutionState {
	var ret []model.ExecutionState
	shardState, ok := jobState.Shards[shardIndex]
	if !ok {
		return ret
	}
	return shardState.Executions
}

func GetFilteredShardStates(jobState model.JobState, filterState model.ExecutionStateType) []model.ExecutionState {
	var ret []model.ExecutionState
	for _, shardState := range FlattenExecutionStates(jobState) { //nolint:gocritic
		if shardState.State == filterState {
			ret = append(ret, shardState)
		}
	}
	return ret
}

func CountVerifiedShardStates(jobState model.JobState) int {
	count := 0
	for _, shardState := range FlattenExecutionStates(jobState) { //nolint:gocritic
		if shardState.VerificationResult.Result {
			count++
		}
	}
	return count
}

func GetCompletedShardStates(jobState model.JobState) []model.ExecutionState {
	return GetFilteredShardStates(jobState, model.ExecutionStateCompleted)
}

// return only shard states that are both complete and verified
func GetCompletedVerifiedShardStates(jobState model.JobState) []model.ExecutionState {
	ret := []model.ExecutionState{}
	for _, shardState := range GetFilteredShardStates(jobState, model.ExecutionStateCompleted) { //nolint:gocritic
		if shardState.VerificationResult.Complete && shardState.VerificationResult.Result && shardState.PublishedResult.CID != "" {
			ret = append(ret, shardState)
		}
	}
	return ret
}

func GetShardStateTotals(shardStates []model.ExecutionState) map[model.ExecutionStateType]int {
	discoveredStateCount := map[model.ExecutionStateType]int{}
	for _, shardState := range shardStates { //nolint:gocritic
		discoveredStateCount[shardState.State]++
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
					e.Msg("Shard failed")
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
		allShardStates := FlattenExecutionStates(jobState)
		discoveredStateCount := GetShardStateTotals(allShardStates)
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
// such as when one node publishes the result before others, or when confidence factor is lower than concurrency.
// for that reason, we consider a job to be in a terminal state when:
// - all executions are in a terminal state
// - shards are in terminal states to account for possible retries
// - the job is in a terminal state to account for possible retries
func WaitForTerminalStates() CheckStatesFunction {
	return func(jobState model.JobState) (bool, error) {
		executionStates := FlattenExecutionStates(jobState)
		for _, executionState := range executionStates {
			if !executionState.State.IsTerminal() {
				return false, nil
			}
		}
		for _, shardState := range jobState.Shards {
			if !shardState.State.IsTerminal() {
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

// if there are > X states then error
func WaitDontExceedCount(count int) CheckStatesFunction {
	return func(jobState model.JobState) (bool, error) {
		allShardStates := FlattenExecutionStates(jobState)
		if len(allShardStates) > count {
			return false, fmt.Errorf("there are more states: %d than expected: %d", len(allShardStates), count)
		}
		return true, nil
	}
}
