package job

import (
	"context"
	"fmt"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/rs/zerolog/log"
)

type JobLoader func(ctx context.Context, id string) (executor.Job, error)
type StateLoader func(ctx context.Context, id string) (executor.JobState, error)

// a function that is given a map of nodeid -> job states
// and will throw an error if anything about that is wrong
type CheckStatesFunction func(executor.JobState) (bool, error)

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

func (resolver *StateResolver) GetShards(ctx context.Context, jobID string) ([]executor.JobShardState, error) {
	jobState, err := resolver.stateLoader(ctx, jobID)
	if err != nil {
		return []executor.JobShardState{}, err
	}
	return FlattenShardStates(jobState), nil
}

func (resolver *StateResolver) StateSummary(ctx context.Context, jobID string) (string, error) {
	jobState, err := resolver.stateLoader(ctx, jobID)
	if err != nil {
		return "", err
	}

	var currentJobState executor.JobStateType
	for _, shardState := range FlattenShardStates(jobState) {
		if shardState.State > currentJobState {
			currentJobState = shardState.State
		}
	}

	return currentJobState.String(), nil
}

func (resolver *StateResolver) ResultSummary(ctx context.Context, jobID string) (string, error) {
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
	completedShards := GetCompletedShardStates(jobState)
	if len(completedShards) == 0 {
		return "", nil
	}
	return fmt.Sprintf("/ipfs/%s", completedShards[0].ResultsID), nil
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
	waiter := &system.FunctionWaiter{
		Name:        "wait for job",
		MaxAttempts: resolver.maxWaitAttempts,
		Delay:       resolver.waitDelay,
		Handler: func() (bool, error) {
			jobState, err := resolver.stateLoader(ctx, jobID)
			if err != nil {
				return false, err
			}

			allOk := true
			for _, checkFunction := range checkJobStateFunctions {
				stepOk, err := checkFunction(jobState)
				if err != nil {
					return false, err
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
			allShardStates := FlattenShardStates(jobState)

			// If all the jobs are in terminal states, then nothing is going
			// to change if we keep polling, so we should exit early.
			allTerminal := len(allShardStates) == totalShards
			for _, shard := range allShardStates {
				if !shard.State.IsTerminal() {
					allTerminal = false
					break
				}
			}
			if allTerminal {
				return false, fmt.Errorf("all jobs are in terminal states and conditions aren't met")
			}
			return false, nil
		},
	}

	return waiter.Wait()
}

// this is an auto wait where we auto calculate how many shard
// sates we expect to see and we use that to pass to WaitForJobStates
func (resolver *StateResolver) WaitUntilComplete(ctx context.Context, jobID string) error {
	job, err := resolver.jobLoader(ctx, jobID)
	if err != nil {
		return err
	}
	totalShards := GetJobTotalExecutionCount(job)
	return resolver.Wait(
		ctx,
		jobID,
		totalShards,
		WaitThrowErrors([]executor.JobStateType{
			executor.JobStateCancelled,
			executor.JobStateError,
		}),
		WaitForJobStates(map[executor.JobStateType]int{
			executor.JobStateComplete: totalShards,
		}),
	)
}

type ResultsShard struct {
	ShardIndex int
	ResultsID  string
}

func (resolver *StateResolver) GetResults(ctx context.Context, jobID string) ([]ResultsShard, error) {
	results := []ResultsShard{}
	job, err := resolver.jobLoader(ctx, jobID)
	if err != nil {
		return results, err
	}
	jobState, err := resolver.stateLoader(ctx, jobID)
	if err != nil {
		return results, err
	}
	totalShards := GetJobTotalShards(job)
	groupedShardResults := GroupShardStates(GetCompletedShardStates(jobState))

	// we have already filtered down to complete results
	// so there must be totalShards entries in the groupedShardResults
	// and it means we have a complete result set
	if len(groupedShardResults) < totalShards {
		return results, fmt.Errorf(
			"job (%s) has not completed yet - %d shards out of %d are complete",
			jobID,
			len(groupedShardResults),
			totalShards,
		)
	}

	// now let's pluck the first result from each shard
	for shardIndex, shardResults := range groupedShardResults {
		// this is a sanity check - there should never be an empty
		// array in the groupedShardResults but just in case
		if len(shardResults) == 0 {
			return results, fmt.Errorf(
				"job (%s) has an empty shard result map at shard index %d",
				jobID,
				shardIndex,
			)
		}

		shardResult := shardResults[0]

		// again this should never happen but just in case
		// a shard result with an empty CID has made it through somehow
		if shardResult.ResultsID == "" {
			return results, fmt.Errorf(
				"job (%s) has a missing results id at shard index %d",
				jobID,
				shardIndex,
			)
		}

		results = append(results, ResultsShard{
			ShardIndex: shardIndex,
			ResultsID:  shardResult.ResultsID,
		})
	}
	return results, nil
}

func FlattenShardStates(jobState executor.JobState) []executor.JobShardState {
	ret := []executor.JobShardState{}
	for _, nodeState := range jobState.Nodes {
		for _, shardState := range nodeState.Shards {
			ret = append(ret, shardState)
		}
	}
	return ret
}

func GetFilteredShardStates(jobState executor.JobState, filterState executor.JobStateType) []executor.JobShardState {
	ret := []executor.JobShardState{}
	for _, shardState := range FlattenShardStates(jobState) {
		if shardState.State == filterState {
			ret = append(ret, shardState)
		}
	}
	return ret
}

func GetCompletedShardStates(jobState executor.JobState) []executor.JobShardState {
	return GetFilteredShardStates(jobState, executor.JobStateComplete)
}

func HasShardReachedCapacity(job executor.Job, jobState executor.JobState, shardIndex int) bool {
	allShards := GroupShardStates(FlattenShardStates(jobState))
	shardStates, ok := allShards[shardIndex]
	if !ok {
		return false
	}

	bidsSeen := 0
	acceptedBidsSeen := 0

	for _, shardState := range shardStates {
		if shardState.State == executor.JobStateBidding {
			bidsSeen++
		} else if shardState.State == executor.JobStateWaiting {
			acceptedBidsSeen++
		}
	}

	if acceptedBidsSeen >= job.Deal.Concurrency {
		return true
	}

	if bidsSeen >= job.Deal.Concurrency*2 {
		return true
	}

	return false
}

// group states by shard index so we can easily iterate over a whole set of them
func GroupShardStates(flatShards []executor.JobShardState) map[int][]executor.JobShardState {
	ret := map[int][]executor.JobShardState{}
	for _, shardState := range flatShards {
		arr, ok := ret[shardState.ShardIndex]
		if !ok {
			arr = []executor.JobShardState{}
		}
		arr = append(arr, shardState)
		ret[shardState.ShardIndex] = arr
	}
	return ret
}

func GetShardStateTotals(shardStates []executor.JobShardState) map[executor.JobStateType]int {
	discoveredStateCount := map[executor.JobStateType]int{}
	for _, shardState := range shardStates {
		discoveredStateCount[shardState.State]++
	}
	return discoveredStateCount
}

// error if there are any errors in any of the states
func WaitThrowErrors(errorStates []executor.JobStateType) CheckStatesFunction {
	return func(jobState executor.JobState) (bool, error) {
		allShardStates := FlattenShardStates(jobState)
		for _, shard := range allShardStates {
			if shard.State.IsError() {
				return false, fmt.Errorf("job has error state %s on node %s", shard.State.String(), shard.NodeID)
			}
		}
		return true, nil
	}
}

// wait for the given number of different states to occur
func WaitForJobStates(requiredStateCounts map[executor.JobStateType]int) CheckStatesFunction {
	return func(jobState executor.JobState) (bool, error) {
		allShardStates := FlattenShardStates(jobState)
		discoveredStateCount := GetShardStateTotals(allShardStates)
		log.Debug().Msgf("WaitForJobShouldHaveStates:\nrequired = %+v,\nactual = %+v\n", requiredStateCounts, discoveredStateCount)
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

// if there are > X states then error
func WaitDontExceedCount(count int) CheckStatesFunction {
	return func(jobState executor.JobState) (bool, error) {
		allShardStates := FlattenShardStates(jobState)
		if len(allShardStates) > count {
			return false, fmt.Errorf("there are more states: %d than expected: %d", len(allShardStates), count)
		}
		return true, nil
	}
}
