# ModelJobEvent

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**api_version** | **str** | APIVersion of the Job | [optional] 
**client_id** | **str** | optional clientID if this is an externally triggered event (like create job) | [optional] 
**deal** | [**ModelDeal**](ModelDeal.md) |  | [optional] 
**event_name** | **int** |  | [optional] 
**event_time** | **str** |  | [optional] 
**job_execution_plan** | [**ModelJobExecutionPlan**](ModelJobExecutionPlan.md) |  | [optional] 
**job_id** | **str** |  | [optional] 
**published_result** | [**ModelStorageSpec**](ModelStorageSpec.md) |  | [optional] 
**run_output** | [**ModelRunCommandResult**](ModelRunCommandResult.md) |  | [optional] 
**sender_public_key** | **list[int]** |  | [optional] 
**shard_index** | **int** | what shard is this event for | [optional] 
**source_node_id** | **str** | the node that emitted this event | [optional] 
**spec** | [**ModelSpec**](ModelSpec.md) |  | [optional] 
**status** | **str** |  | [optional] 
**target_node_id** | **str** | the node that this event is for e.g. \&quot;AcceptJobBid\&quot; was emitted by Requester but it targeting compute node | [optional] 
**verification_proposal** | **list[int]** |  | [optional] 
**verification_result** | [**ModelVerificationResult**](ModelVerificationResult.md) |  | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

