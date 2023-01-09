# JobEvent

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**api_version** | **str** | APIVersion of the Job | [optional] 
**client_id** | **str** | optional clientID if this is an externally triggered event (like create job) | [optional] 
**deal** | [**JobEventDeal**](JobEventDeal.md) |  | [optional] 
**event_name** | [**JobEventType**](JobEventType.md) |  | [optional] 
**event_time** | **str** |  | [optional] 
**job_execution_plan** | [**JobEventJobExecutionPlan**](JobEventJobExecutionPlan.md) |  | [optional] 
**job_id** | **str** |  | [optional] 
**published_result** | [**StorageSpec**](StorageSpec.md) |  | [optional] 
**run_output** | [**JobEventRunOutput**](JobEventRunOutput.md) |  | [optional] 
**sender_public_key** | **list[int]** |  | [optional] 
**shard_index** | **int** | what shard is this event for | [optional] 
**source_node_id** | **str** | the node that emitted this event | [optional] 
**spec** | [**JobEventJobExecutionPlan**](JobEventJobExecutionPlan.md) |  | [optional] 
**status** | **str** |  | [optional] 
**target_node_id** | **str** | the node that this event is for e.g. \&quot;AcceptJobBid\&quot; was emitted by Requester but it targeting compute node | [optional] 
**verification_proposal** | **list[int]** |  | [optional] 
**verification_result** | [**VerificationResult**](VerificationResult.md) |  | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


