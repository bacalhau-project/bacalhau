package shared

import (
	"fmt"

	model "github.com/filecoin-project/bacalhau/pkg/model/v1beta1"
)

func UpdateShardState(
	nodeID string,
	shardIndex int,
	jobState *model.JobState,
	update model.JobShardState,
) error {
	nodeState, ok := jobState.Nodes[nodeID]
	if !ok {
		nodeState = model.JobNodeState{
			Shards: map[int]model.JobShardState{},
		}
	}
	shardState, ok := nodeState.Shards[shardIndex]
	if !ok {
		shardState = model.JobShardState{
			NodeID:     nodeID,
			ShardIndex: shardIndex,
		}
	}

	if update.State < shardState.State {
		return fmt.Errorf("cannot update shard state to %s as current state is %s. [NodeID: %s, ShardID: %d]",
			update.State, shardState.State, nodeID, shardIndex)
	}

	shardState.State = update.State
	if update.Status != "" {
		shardState.Status = update.Status
	}

	if update.RunOutput != nil {
		shardState.RunOutput = update.RunOutput
	}

	if len(update.VerificationProposal) != 0 {
		shardState.VerificationProposal = update.VerificationProposal
	}

	if update.VerificationResult.Complete {
		shardState.VerificationResult = update.VerificationResult
	}

	if update.ExecutionID != "" {
		shardState.ExecutionID = update.ExecutionID
	}

	if model.IsValidStorageSourceType(update.PublishedResult.StorageSource) {
		shardState.PublishedResult = update.PublishedResult
	}

	nodeState.Shards[shardIndex] = shardState
	jobState.Nodes[nodeID] = nodeState
	return nil
}
