# ExecutionState

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**compute_reference** | **str** | Compute node reference for this job execution | [optional]
**create_time** | **str** | CreateTime is the time when the job was created. | [optional]
**job_id** | **str** | JobID the job id | [optional]
**node_id** | **str** | which node is running this execution | [optional]
**published_results** | [**StorageSpec**](StorageSpec.md) |  | [optional]
**run_output** | **AllOfExecutionStateRunOutput** | RunOutput of the job | [optional]
**state** | **AllOfExecutionStateState** | State is the current state of the execution | [optional]
**status** | **str** | an arbitrary status message | [optional]
**update_time** | **str** | UpdateTime is the time when the job state was last updated. | [optional]
**verification_proposal** | **list[int]** | the proposed results for this execution this will be resolved by the verifier somehow | [optional]
**verification_result** | [**VerificationResult**](VerificationResult.md) |  | [optional]
**version** | **int** | Version is the version of the job state. It is incremented every time the job state is updated. | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
