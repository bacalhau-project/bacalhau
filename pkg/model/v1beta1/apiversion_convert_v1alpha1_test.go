package v1beta1

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/model/v1alpha1"
	"github.com/go-test/deep"
	"github.com/stretchr/testify/require"
)

func getJobs() (v1alpha1.Job, Job) {
	nodeID := "test-node"
	shardIndex := 0
	concurrency := 3
	jobID := "test-job"
	clientID := "test-client"
	CID := "QmX"
	entrypoint := "hello"
	stdout := "oranges"
	status := "pineapples"
	params := []string{"world"}
	createdAt := time.Now()

	v1alpha := v1alpha1.Job{
		APIVersion:      V1alpha1.String(),
		ID:              jobID,
		RequesterNodeID: nodeID,
		ClientID:        clientID,
		CreatedAt:       createdAt,
		Spec: v1alpha1.Spec{
			Engine: v1alpha1.EngineWasm,
			Wasm: v1alpha1.JobSpecWasm{
				EntryPoint: entrypoint,
				Parameters: params,
			},
			Inputs: []v1alpha1.StorageSpec{
				{
					StorageSource: v1alpha1.StorageSourceIPFS,
					CID:           CID,
				},
			},
		},
		Deal: v1alpha1.Deal{
			Concurrency: concurrency,
		},
		ExecutionPlan: v1alpha1.JobExecutionPlan{
			TotalShards: concurrency,
		},
		State: v1alpha1.JobState{
			Nodes: map[string]v1alpha1.JobNodeState{
				nodeID: {
					Shards: map[int]v1alpha1.JobShardState{
						shardIndex: {
							NodeID:     nodeID,
							ShardIndex: shardIndex,
							State:      v1alpha1.JobStateBidding,
							Status:     status,
							PublishedResult: v1alpha1.StorageSpec{
								StorageSource: v1alpha1.StorageSourceIPFS,
								CID:           CID,
							},
							RunOutput: &v1alpha1.RunCommandResult{
								STDOUT: stdout,
							},
						},
					},
				},
			},
		},
		Events: []v1alpha1.JobEvent{
			{
				APIVersion:   V1alpha1.String(),
				JobID:        jobID,
				ShardIndex:   shardIndex,
				ClientID:     clientID,
				SourceNodeID: nodeID,
				TargetNodeID: nodeID,
				EventName:    v1alpha1.JobEventBid,
				RunOutput: &v1alpha1.RunCommandResult{
					STDOUT: stdout,
				},
			},
		},
		LocalEvents: []v1alpha1.JobLocalEvent{
			{
				EventName:    v1alpha1.JobLocalEventBid,
				JobID:        jobID,
				ShardIndex:   shardIndex,
				TargetNodeID: nodeID,
			},
		},
	}

	latest := Job{
		APIVersion: APIVersionLatest().String(),
		Metadata: Metadata{
			ID:        jobID,
			CreatedAt: createdAt,
			ClientID:  clientID,
		},
		Spec: Spec{
			Engine: EngineWasm,
			Wasm: JobSpecWasm{
				EntryPoint: entrypoint,
				Parameters: params,
			},
			Inputs: []StorageSpec{
				{
					StorageSource: StorageSourceIPFS,
					CID:           CID,
				},
			},
			Deal: Deal{
				Concurrency: concurrency,
			},
			ExecutionPlan: JobExecutionPlan{
				TotalShards: concurrency,
			},
		},
		Status: JobStatus{
			State: JobState{
				Nodes: map[string]JobNodeState{
					nodeID: {
						Shards: map[int]JobShardState{
							shardIndex: {
								NodeID:     nodeID,
								ShardIndex: shardIndex,
								State:      JobStateBidding,
								Status:     status,
								PublishedResult: StorageSpec{
									StorageSource: StorageSourceIPFS,
									CID:           CID,
								},
								RunOutput: &RunCommandResult{
									STDOUT: stdout,
								},
							},
						},
					},
				},
			},
			Events: []JobEvent{
				{
					APIVersion:   APIVersionLatest().String(),
					JobID:        jobID,
					ShardIndex:   shardIndex,
					ClientID:     clientID,
					SourceNodeID: nodeID,
					TargetNodeID: nodeID,
					EventName:    JobEventBid,
					RunOutput: &RunCommandResult{
						STDOUT: stdout,
					},
				},
			},
			LocalEvents: []JobLocalEvent{
				{
					EventName:    JobLocalEventBid,
					JobID:        jobID,
					ShardIndex:   shardIndex,
					TargetNodeID: nodeID,
				},
			},
			Requester: JobRequester{
				RequesterNodeID: nodeID,
			},
		},
	}

	return v1alpha, latest
}

func TestParseAPIVersion(t *testing.T) {
	v, err := ParseAPIVersion("V1beta1")
	require.NoError(t, err)
	require.Equal(t, v, V1beta1)
}

func TestConvertV1Alpha1_Job(t *testing.T) {
	oldData, compareData := getJobs()
	oldJSONString, err := json.Marshal(oldData)
	require.NoError(t, err)
	newData, err := APIVersionParseJob(V1alpha1.String(), string(oldJSONString))
	require.NoError(t, err)
	if diff := deep.Equal(newData, compareData); diff != nil {
		t.Error(diff)
	}
}
