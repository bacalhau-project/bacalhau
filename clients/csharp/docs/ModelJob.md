# bacalhau-client.Model.ModelJob
## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**APIVersion** | **string** |  | [optional] 
**ClientID** | **string** | The ID of the client that created this job. | [optional] 
**CreatedAt** | **string** | Time the job was submitted to the bacalhau network. | [optional] 
**Deal** | [**ModelDeal**](ModelDeal.md) |  | [optional] 
**ExecutionPlan** | [**ModelJobExecutionPlan**](ModelJobExecutionPlan.md) |  | [optional] 
**ID** | **string** | The unique global ID of this job in the bacalhau network. | [optional] 
**JobEvents** | [**List&lt;ModelJobEvent&gt;**](ModelJobEvent.md) | All events associated with the job | [optional] 
**JobState** | [**ModelJobState**](ModelJobState.md) |  | [optional] 
**LocalJobEvents** | [**List&lt;ModelJobLocalEvent&gt;**](ModelJobLocalEvent.md) | All local events associated with the job | [optional] 
**RequesterNodeID** | **string** | The ID of the requester node that owns this job. | [optional] 
**RequesterPublicKey** | **List&lt;int?&gt;** | The public key of the Requester node that created this job This can be used to encrypt messages back to the creator | [optional] 
**Spec** | [**ModelSpec**](ModelSpec.md) |  | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

