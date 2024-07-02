# Execution

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**allocated_resources** | **AllOfExecutionAllocatedResources** | AllocatedResources is the total resources allocated for the execution tasks. | [optional] 
**compute_state** | **AllOfExecutionComputeState** | ComputeState observed state of the execution on the compute node | [optional] 
**create_time** | **int** | CreateTime is the time the execution has finished scheduling and been verified by the plan applier. | [optional] 
**desired_state** | **AllOfExecutionDesiredState** | DesiredState of the execution on the compute node | [optional] 
**eval_id** | **str** | ID of the evaluation that generated this execution | [optional] 
**followup_eval_id** | **str** | FollowupEvalID captures a follow up evaluation created to handle a failed execution that can be rescheduled in the future | [optional] 
**id** | **str** | ID of the execution (UUID) | [optional] 
**job** | **AllOfExecutionJob** | TODO: evaluate using a copy of the job instead of a pointer | [optional] 
**job_id** | **str** | Job is the parent job of the task being allocated. This is copied at execution time to avoid issues if the job definition is updated. | [optional] 
**modify_time** | **int** | ModifyTime is the time the execution was last updated. | [optional] 
**name** | **str** | Name is a logical name of the execution. | [optional] 
**namespace** | **str** | Namespace is the namespace the execution is created in | [optional] 
**next_execution** | **str** | NextExecution is the execution that this execution is being replaced by | [optional] 
**node_id** | **str** | NodeID is the node this is being placed on | [optional] 
**previous_execution** | **str** | PreviousExecution is the execution that this execution is replacing | [optional] 
**published_result** | **AllOfExecutionPublishedResult** | the published results for this execution | [optional] 
**revision** | **int** | Revision is increment each time the execution is updated. | [optional] 
**run_output** | **AllOfExecutionRunOutput** | RunOutput is the output of the run command TODO: evaluate removing this from execution spec in favour of calling &#x60;bacalhau logs&#x60; | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

