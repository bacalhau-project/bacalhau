package localdb

// TODO: This is copied from an older version of job/state package. It is a temporary solution until
//  jobStore and localdb are merged. #1943

import (
	"context"

	model "github.com/filecoin-project/bacalhau/pkg/model/v1beta1"
	"github.com/filecoin-project/bacalhau/pkg/system"
)

type JobLoader func(ctx context.Context, id string) (*model.Job, error)
type StateLoader func(ctx context.Context, id string) (model.JobState, error)

func GetStateResolver(db LocalDB) *StateResolver {
	return NewStateResolver(
		db.GetJob,
		db.GetJobState,
	)
}

type StateResolver struct {
	jobLoader   JobLoader
	stateLoader StateLoader
}

func NewStateResolver(
	jobLoader JobLoader,
	stateLoader StateLoader,
) *StateResolver {
	return &StateResolver{
		jobLoader:   jobLoader,
		stateLoader: stateLoader,
	}
}

func (resolver *StateResolver) GetJobState(ctx context.Context, id string) (model.JobState, error) {
	return resolver.stateLoader(ctx, id)
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

func FlattenShardStates(jobState model.JobState) []model.JobShardState {
	ret := []model.JobShardState{}
	for _, nodeState := range jobState.Nodes {
		for _, shardState := range nodeState.Shards { //nolint:gocritic
			ret = append(ret, shardState)
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
