# ModelJobEvent

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**api_version** | **string** | APIVersion of the Job | [optional] 
**client_id** | **string** | optional clientID if this is an externally triggered event (like create job) | [optional] 
**deal** | [**\Swagger\Client\Model\ModelDeal**](ModelDeal.md) |  | [optional] 
**event_name** | **int** |  | [optional] 
**event_time** | **string** |  | [optional] 
**job_execution_plan** | [**\Swagger\Client\Model\ModelJobExecutionPlan**](ModelJobExecutionPlan.md) |  | [optional] 
**job_id** | **string** |  | [optional] 
**published_result** | [**\Swagger\Client\Model\ModelStorageSpec**](ModelStorageSpec.md) |  | [optional] 
**run_output** | [**\Swagger\Client\Model\ModelRunCommandResult**](ModelRunCommandResult.md) |  | [optional] 
**sender_public_key** | **int[]** |  | [optional] 
**shard_index** | **int** | what shard is this event for | [optional] 
**source_node_id** | **string** | the node that emitted this event | [optional] 
**spec** | [**\Swagger\Client\Model\ModelSpec**](ModelSpec.md) |  | [optional] 
**status** | **string** |  | [optional] 
**target_node_id** | **string** | the node that this event is for e.g. \&quot;AcceptJobBid\&quot; was emitted by Requester but it targeting compute node | [optional] 
**verification_proposal** | **int[]** |  | [optional] 
**verification_result** | [**\Swagger\Client\Model\ModelVerificationResult**](ModelVerificationResult.md) |  | [optional] 

[[Back to Model list]](../../README.md#documentation-for-models) [[Back to API list]](../../README.md#documentation-for-api-endpoints) [[Back to README]](../../README.md)

