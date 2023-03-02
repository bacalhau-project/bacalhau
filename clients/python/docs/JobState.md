# JobState

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**create_time** | **str** | CreateTime is the time when the job was created. | [optional]
**job_id** | **str** | JobID is the unique identifier for the job | [optional]
**shards** | [**dict(str, ShardState)**](ShardState.md) | Shards is a map of shard index to shard state. The number of shards are fixed at the time of job creation. | [optional]
**state** | **AllOfJobStateState** | State is the current state of the job | [optional]
**timeout_at** | **str** | TimeoutAt is the time when the job will be timed out if it is not completed. | [optional]
**update_time** | **str** | UpdateTime is the time when the job state was last updated. | [optional]
**version** | **int** | Version is the version of the job state. It is incremented every time the job state is updated. | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
