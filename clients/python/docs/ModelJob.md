# ModelJob

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**api_version** | **str** |  | [optional] 
**client_id** | **str** | The ID of the client that created this job. | [optional] 
**created_at** | **str** | Time the job was submitted to the bacalhau network. | [optional] 
**deal** | [**ModelDeal**](ModelDeal.md) |  | [optional] 
**execution_plan** | [**ModelJobExecutionPlan**](ModelJobExecutionPlan.md) |  | [optional] 
**id** | **str** | The unique global ID of this job in the bacalhau network. | [optional] 
**job_events** | [**list[ModelJobEvent]**](ModelJobEvent.md) | All events associated with the job | [optional] 
**job_state** | [**ModelJobState**](ModelJobState.md) |  | [optional] 
**local_job_events** | [**list[ModelJobLocalEvent]**](ModelJobLocalEvent.md) | All local events associated with the job | [optional] 
**requester_node_id** | **str** | The ID of the requester node that owns this job. | [optional] 
**requester_public_key** | **list[int]** | The public key of the Requester node that created this job This can be used to encrypt messages back to the creator | [optional] 
**spec** | [**ModelSpec**](ModelSpec.md) |  | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

