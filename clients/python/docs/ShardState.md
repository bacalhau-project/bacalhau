# ShardState

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**create_time** | **str** | CreateTime is the time when the shard was created, which is the same as the job creation time. | [optional]
**executions** | [**list[ExecutionState]**](ExecutionState.md) | Executions is a list of executions of the shard across the nodes. A new execution is created when a node is selected to execute the shard, and a node can have multiple executions for the same shard due to retries, but there can only be a single active execution per node at any given time. | [optional]
**job_id** | **str** | JobID is the unique identifier for the job | [optional]
**shard_index** | **int** | ShardIndex is the index of the shard in the job | [optional]
**state** | **AllOfShardStateState** | State is the current state of the shard | [optional]
**update_time** | **str** | UpdateTime is the time when the shard state was last updated. | [optional]
**version** | **int** | Version is the version of the shard state. It is incremented every time the shard state is updated. | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
