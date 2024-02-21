# ExecutionState

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**compute_reference** | **str** | Compute node reference for this job execution | [optional]
**create_time** | **str** | CreateTime is the time when the job was created. | [optional]
**desired_state** | **AllOfExecutionStateDesiredState** | DesiredState is the desired state of the execution | [optional]
**job_id** | **str** | JobID the job id | [optional]
**node_id** | **str** | which node is running this execution | [optional]
**published_results** | **AllOfExecutionStatePublishedResults** | the published results for this execution | [optional]
**run_output** | **AllOfExecutionStateRunOutput** | RunOutput of the job | [optional]
**state** | **AllOfExecutionStateState** | State is the current state of the execution | [optional]
**status** | **str** | an arbitrary status message | [optional]
**update_time** | **str** | UpdateTime is the time when the job state was last updated. | [optional]
**version** | **int** | Version is the version of the job state. It is incremented every time the job state is updated. | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
