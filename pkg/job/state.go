package job

import (
	"context"
	"fmt"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/executor"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/rs/zerolog/log"
)

type StateLoader func(ctx context.Context, id string) (executor.JobState, error)

// a function that is given a map of nodeid -> job states
// and will throw an error if anything about that is wrong
type CheckStatesFunction func(executor.JobState) (bool, error)

type StateResolver struct {
	job             executor.Job
	ctx             context.Context
	loader          StateLoader
	maxWaitAttempts int
	waitDelay       time.Duration
}

func NewStateResolver(
	job executor.Job,
	ctx context.Context,
	stateLoader StateLoader,
) *StateResolver {
	return &StateResolver{
		job:             job,
		ctx:             ctx,
		loader:          stateLoader,
		maxWaitAttempts: 100,
		waitDelay:       time.Second * 1,
	}
}

func (resolver *StateResolver) GetShards() ([]executor.JobShardState, error) {
	jobState, err := resolver.loader(resolver.ctx, resolver.job.ID)
	if err != nil {
		return []executor.JobShardState{}, err
	}
	return FlattenShardStates(jobState), nil
}

// this will return a list of results groups by shard index
// it will pick the first "complete" state for each shard
// if there are any shards missing (because they errored or are not ready yet)
// then this will error so it's important that you check the job has completed
// before calling this
// TODO: this should probably be part of the verifier interface
func (resolver *StateResolver) GetResults() ([]string, error) {
	ret := []string{}
	jobState, err := resolver.loader(resolver.ctx, resolver.job.ID)
	if err != nil {
		return ret, err
	}
	allShardStates := FlattenShardStates(jobState)
	for _, shard := range allShardStates {
		if shard.ResultsID != "" {
			ret = append(ret, shard.ResultsID)
		}
	}
	return ret, nil
}

func (resolver *StateResolver) StateSummary() (string, error) {
	_, err := resolver.loader(resolver.ctx, resolver.job.ID)
	if err != nil {
		return "", err
	}
	return "state summary", nil
}

func (resolver *StateResolver) ResultSummary() (string, error) {
	return "result summary", nil
}

func (resolver *StateResolver) Wait(
	ctx context.Context,
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
			jobState, err := resolver.loader(resolver.ctx, resolver.job.ID)
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
func (resolver *StateResolver) WaitUntilComplete(ctx context.Context) error {
	totalShards := GetJobTotalExecutionCount(resolver.job)
	return resolver.Wait(
		ctx,
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
