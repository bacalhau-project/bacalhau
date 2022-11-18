# SwaggerClient::ModelJob

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**api_version** | **String** |  | [optional] 
**client_id** | **String** | The ID of the client that created this job. | [optional] 
**created_at** | **String** | Time the job was submitted to the bacalhau network. | [optional] 
**deal** | [**ModelDeal**](ModelDeal.md) |  | [optional] 
**execution_plan** | [**ModelJobExecutionPlan**](ModelJobExecutionPlan.md) |  | [optional] 
**id** | **String** | The unique global ID of this job in the bacalhau network. | [optional] 
**job_events** | [**Array&lt;ModelJobEvent&gt;**](ModelJobEvent.md) | All events associated with the job | [optional] 
**job_state** | [**ModelJobState**](ModelJobState.md) |  | [optional] 
**local_job_events** | [**Array&lt;ModelJobLocalEvent&gt;**](ModelJobLocalEvent.md) | All local events associated with the job | [optional] 
**requester_node_id** | **String** | The ID of the requester node that owns this job. | [optional] 
**requester_public_key** | **Array&lt;Integer&gt;** | The public key of the Requester node that created this job This can be used to encrypt messages back to the creator | [optional] 
**spec** | [**ModelSpec**](ModelSpec.md) |  | [optional] 

