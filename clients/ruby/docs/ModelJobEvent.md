# SwaggerClient::ModelJobEvent

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**api_version** | **String** | APIVersion of the Job | [optional] 
**client_id** | **String** | optional clientID if this is an externally triggered event (like create job) | [optional] 
**deal** | [**ModelDeal**](ModelDeal.md) |  | [optional] 
**event_name** | **Integer** |  | [optional] 
**event_time** | **String** |  | [optional] 
**job_execution_plan** | [**ModelJobExecutionPlan**](ModelJobExecutionPlan.md) |  | [optional] 
**job_id** | **String** |  | [optional] 
**published_result** | [**ModelStorageSpec**](ModelStorageSpec.md) |  | [optional] 
**run_output** | [**ModelRunCommandResult**](ModelRunCommandResult.md) |  | [optional] 
**sender_public_key** | **Array&lt;Integer&gt;** |  | [optional] 
**shard_index** | **Integer** | what shard is this event for | [optional] 
**source_node_id** | **String** | the node that emitted this event | [optional] 
**spec** | [**ModelSpec**](ModelSpec.md) |  | [optional] 
**status** | **String** |  | [optional] 
**target_node_id** | **String** | the node that this event is for e.g. \&quot;AcceptJobBid\&quot; was emitted by Requester but it targeting compute node | [optional] 
**verification_proposal** | **Array&lt;Integer&gt;** |  | [optional] 
**verification_result** | [**ModelVerificationResult**](ModelVerificationResult.md) |  | [optional] 

