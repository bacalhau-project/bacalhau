# ModelJob

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**api_version** | **string** |  | [optional] 
**client_id** | **string** | The ID of the client that created this job. | [optional] 
**created_at** | **string** | Time the job was submitted to the bacalhau network. | [optional] 
**deal** | [**\Swagger\Client\Model\ModelDeal**](ModelDeal.md) |  | [optional] 
**execution_plan** | [**\Swagger\Client\Model\ModelJobExecutionPlan**](ModelJobExecutionPlan.md) |  | [optional] 
**id** | **string** | The unique global ID of this job in the bacalhau network. | [optional] 
**job_events** | [**\Swagger\Client\Model\ModelJobEvent[]**](ModelJobEvent.md) | All events associated with the job | [optional] 
**job_state** | [**\Swagger\Client\Model\ModelJobState**](ModelJobState.md) |  | [optional] 
**local_job_events** | [**\Swagger\Client\Model\ModelJobLocalEvent[]**](ModelJobLocalEvent.md) | All local events associated with the job | [optional] 
**requester_node_id** | **string** | The ID of the requester node that owns this job. | [optional] 
**requester_public_key** | **int[]** | The public key of the Requester node that created this job This can be used to encrypt messages back to the creator | [optional] 
**spec** | [**\Swagger\Client\Model\ModelSpec**](ModelSpec.md) |  | [optional] 

[[Back to Model list]](../../README.md#documentation-for-models) [[Back to API list]](../../README.md#documentation-for-api-endpoints) [[Back to README]](../../README.md)

