# JobState

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**create_time** | **str** | CreateTime is the time when the job was created. | [optional]
**executions** | [**list[ExecutionState]**](ExecutionState.md) | Executions is a list of executions of the job across the nodes. A new execution is created when a node is selected to execute the job, and a node can have multiple executions for the same job due to retries, but there can only be a single active execution per node at any given time. | [optional]
**job_id** | **str** | JobID is the unique identifier for the job | [optional]
**state** | **AllOfJobStateState** | State is the current state of the job | [optional]
**timeout_at** | **str** | TimeoutAt is the time when the job will be timed out if it is not completed. | [optional]
**update_time** | **str** | UpdateTime is the time when the job state was last updated. | [optional]
**version** | **int** | Version is the version of the job state. It is incremented every time the job state is updated. | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
