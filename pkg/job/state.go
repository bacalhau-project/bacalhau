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

type JobLoader func(ctx context.Context, id string) (*model.Job, error)
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

func (resolver *StateResolver) GetJob(ctx context.Context, id string) (*model.Job, error) {
	return resolver.jobLoader(ctx, id)
}

func (resolver *StateResolver) GetJobState(ctx context.Context, id string) (model.JobState, error) {
	return resolver.stateLoader(ctx, id)
}

func (resolver *StateResolver) SetWaitTime(maxWaitAttempts int, delay time.Duration) {
	resolver.maxWaitAttempts = maxWaitAttempts
	resolver.waitDelay = delay
}

func (resolver *StateResolver) GetShards(ctx context.Context, jobID string) ([]model.JobShardState, error) {
	jobState, err := resolver.stateLoader(ctx, jobID)
	if err != nil {
		return []model.JobShardState{}, err
	}
	return FlattenShardStates(jobState), nil
}

func (resolver *StateResolver) StateSummary(ctx context.Context, jobID string) (string, error) {
	ctx, span := system.GetTracer().Start(ctx, "pkg/job.StateSummary")
	defer span.End()
	system.AddJobIDFromBaggageToSpan(ctx, span)

	jobState, err := resolver.stateLoader(ctx, jobID)
	if err != nil {
		return "", err
	}

	var currentJobState model.JobStateType
	for _, shardState := range FlattenShardStates(jobState) { //nolint:gocritic
		if shardState.State > currentJobState {
			currentJobState = shardState.State
		}
	}

	return currentJobState.String(), nil
}

func (resolver *StateResolver) VerifiedSummary(ctx context.Context, jobID string) (string, error) {
	ctx, span := system.GetTracer().Start(ctx, "pkg/job.VerifiedSummary")
	defer span.End()
	system.AddJobIDFromBaggageToSpan(ctx, span)

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
	ctx, span := system.GetTracer().Start(ctx, "pkg/job.ResultSummary")
	defer span.End()
	system.AddJobIDFromBaggageToSpan(ctx, span)

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
	// this is the total number of expected states
	// used to quit early if we've not matched our checkJobStateFunctions
	// but all of the loaded states are terminal
	// this number is concurrency * total batches
	totalShards int,
	checkJobStateFunctions ...CheckStatesFunction,
) error {
	return resolver.WaitWithOptions(ctx, WaitOptions{
		JobID:       jobID,
		TotalShards: totalShards,
	}, checkJobStateFunctions...)
}

type WaitOptions struct {
	// the job we are waiting for
	JobID string
	// this is the total number of expected states
	// used to quit early if we've not matched our checkJobStateFunctions
	// but all of the loaded states are terminal
	// this number is concurrency * total batches
	TotalShards int
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
			// let's see if we can quiet early because all expectedd states are
			// in terminal state
			allTerminal, err := WaitForTerminalStates(options.TotalShards)(jobState)
			if err != nil {
				return false, err
			}

			// If all the jobs are in terminal states, then nothing is going
			// to change if we keep polling, so we should exit early.
			if allTerminal && !options.AllowAllTerminal {
				return false, fmt.Errorf("all jobs are in terminal states and conditions aren't met")
			}
			return false, nil
		},
	}

	return waiter.Wait(ctx)
}

// this is an auto wait where we auto calculate how many shard
// states we expect to see and we use that to pass to WaitForJobStates
func (resolver *StateResolver) WaitUntilComplete(ctx context.Context, jobID string) error {
	ctx, span := system.GetTracer().Start(ctx, "pkg/job.WaitUntilComplete")
	defer span.End()
	system.AddJobIDFromBaggageToSpan(ctx, span)

	job, err := resolver.jobLoader(ctx, jobID)
	if err != nil {
		return err
	}
	totalShards := GetJobTotalExecutionCount(job)
	return resolver.Wait(
		ctx,
		jobID,
		totalShards,
		WaitThrowErrors([]model.JobStateType{
			model.JobStateError,
		}),
		WaitForJobStates(map[model.JobStateType]int{
			model.JobStateCompleted: totalShards,
		}),
	)
}

func (resolver *StateResolver) GetResults(ctx context.Context, jobID string) ([]model.PublishedResult, error) {
	ctx, span := system.GetTracer().Start(ctx, "pkg/job.GetResults")
	defer span.End()
	system.AddJobIDFromBaggageToSpan(ctx, span)

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
	shardStates []model.JobShardState,
	concurrency int,
) (bool, error)

// iterate each shard and pass off []model.JobShardState to the given function
// every shard must return true for this function to return true
// this is useful for example to say "do we have enough to begin verification"
func (resolver *StateResolver) CheckShardStates(
	ctx context.Context,
	shard model.JobShard,
	shardStateChecker ShardStateChecker,
) (bool, error) {
	ctx, span := system.GetTracer().Start(ctx, "pkg/job.CheckShardStates")
	defer span.End()
	system.AddJobIDFromBaggageToSpan(ctx, span)

	jobState, err := resolver.stateLoader(ctx, shard.Job.ID)
	if err != nil {
		return false, err
	}

	concurrency := int(math.Max(float64(shard.Job.Deal.Concurrency), 1))
	shardStates := GetStatesForShardIndex(jobState, shard.Index)
	if len(shardStates) == 0 {
		return false, fmt.Errorf("job (%s) has no shard state for shard index %d", shard.Job.ID, shard.Index)
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

func FlattenShardStates(jobState model.JobState) []model.JobShardState {
	ret := []model.JobShardState{}
	for _, nodeState := range jobState.Nodes {
		for _, shardState := range nodeState.Shards { //nolint:gocritic
			ret = append(ret, shardState)
		}
	}
	return ret
}

func GetStatesForShardIndex(jobState model.JobState, shardIndex int) []model.JobShardState {
	ret := []model.JobShardState{}
	for _, nodeState := range jobState.Nodes {
		for _, shardState := range nodeState.Shards { //nolint:gocritic
			if shardState.ShardIndex == shardIndex {
				ret = append(ret, shardState)
			}
		}
	}
	return ret
}

func GetFilteredShardStates(jobState model.JobState, filterState model.JobStateType) []model.JobShardState {
	ret := []model.JobShardState{}
	for _, shardState := range FlattenShardStates(jobState) { //nolint:gocritic
		if shardState.State == filterState {
			ret = append(ret, shardState)
		}
	}
	return ret
}

func CountVerifiedShardStates(jobState model.JobState) int {
	count := 0
	for _, shardState := range FlattenShardStates(jobState) { //nolint:gocritic
		if shardState.VerificationResult.Result {
			count++
		}
	}
	return count
}

func GetCompletedShardStates(jobState model.JobState) []model.JobShardState {
	return GetFilteredShardStates(jobState, model.JobStateCompleted)
}

// return only shard states that are both complete and verified
func GetCompletedVerifiedShardStates(jobState model.JobState) []model.JobShardState {
	ret := []model.JobShardState{}
	for _, shardState := range GetFilteredShardStates(jobState, model.JobStateCompleted) { //nolint:gocritic
		if shardState.VerificationResult.Complete && shardState.VerificationResult.Result && shardState.PublishedResult.CID != "" {
			ret = append(ret, shardState)
		}
	}
	return ret
}

func HasShardReachedCapacity(ctx context.Context, j *model.Job, jobState model.JobState, shardIndex int) bool {
	ctx, span := system.GetTracer().Start(ctx, "pkg/computenode.HasShardReachedCapacity")
	defer span.End()

	system.AddJobIDFromBaggageToSpan(ctx, span)
	system.AddNodeIDFromBaggageToSpan(ctx, span)

	allShards := GroupShardStates(FlattenShardStates(jobState))
	shardStates, ok := allShards[shardIndex]
	if !ok {
		return false
	}

	acceptedBidsSeen := 0

	for _, shardState := range shardStates { //nolint:gocritic
		if shardState.State.HasPassedBidAcceptedStage() {
			acceptedBidsSeen++
		}
	}

	return acceptedBidsSeen >= j.Deal.Concurrency
}

// group states by shard index so we can easily iterate over a whole set of them
func GroupShardStates(flatShards []model.JobShardState) map[int][]model.JobShardState {
	ret := map[int][]model.JobShardState{}
	for _, shardState := range flatShards { //nolint:gocritic
		arr, ok := ret[shardState.ShardIndex]
		if !ok {
			arr = []model.JobShardState{}
		}
		arr = append(arr, shardState)
		ret[shardState.ShardIndex] = arr
	}
	return ret
}

func GetShardStateTotals(shardStates []model.JobShardState) map[model.JobStateType]int {
	discoveredStateCount := map[model.JobStateType]int{}
	for _, shardState := range shardStates { //nolint:gocritic
		discoveredStateCount[shardState.State]++
	}
	return discoveredStateCount
}

// error if there are any errors in any of the states
func WaitThrowErrors(errorStates []model.JobStateType) CheckStatesFunction {
	return func(jobState model.JobState) (bool, error) {
		allShardStates := FlattenShardStates(jobState)
		for _, shard := range allShardStates { //nolint:gocritic
			for _, errorState := range errorStates {
				if shard.State == errorState {
					return false, fmt.Errorf("job has error state %s on node %s (%s)", shard.State.String(), shard.NodeID, shard.Status)
				}
			}
		}
		return true, nil
	}
}

// wait for the given number of different states to occur
func WaitForJobStates(requiredStateCounts map[model.JobStateType]int) CheckStatesFunction {
	return func(jobState model.JobState) (bool, error) {
		allShardStates := FlattenShardStates(jobState)
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

func WaitForTerminalStates(totalShards int) CheckStatesFunction {
	return func(jobState model.JobState) (bool, error) {
		allShardStates := FlattenShardStates(jobState)
		if len(allShardStates) < totalShards {
			return false, nil
		}
		for _, shard := range allShardStates { //nolint:gocritic
			if !shard.State.IsTerminal() {
				return false, nil
			}
		}
		return true, nil
	}
}

// if there are > X states then error
func WaitDontExceedCount(count int) CheckStatesFunction {
	return func(jobState model.JobState) (bool, error) {
		allShardStates := FlattenShardStates(jobState)
		if len(allShardStates) > count {
			return false, fmt.Errorf("there are more states: %d than expected: %d", len(allShardStates), count)
		}
		return true, nil
	}
}
