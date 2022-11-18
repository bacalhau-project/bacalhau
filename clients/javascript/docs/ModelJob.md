# BacalhauClient.ModelJob

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**aPIVersion** | **String** |  | [optional] 
**clientID** | **String** | The ID of the client that created this job. | [optional] 
**createdAt** | **String** | Time the job was submitted to the bacalhau network. | [optional] 
**deal** | [**ModelDeal**](ModelDeal.md) |  | [optional] 
**executionPlan** | [**ModelJobExecutionPlan**](ModelJobExecutionPlan.md) |  | [optional] 
**ID** | **String** | The unique global ID of this job in the bacalhau network. | [optional] 
**jobEvents** | [**[ModelJobEvent]**](ModelJobEvent.md) | All events associated with the job | [optional] 
**jobState** | [**ModelJobState**](ModelJobState.md) |  | [optional] 
**localJobEvents** | [**[ModelJobLocalEvent]**](ModelJobLocalEvent.md) | All local events associated with the job | [optional] 
**requesterNodeID** | **String** | The ID of the requester node that owns this job. | [optional] 
**requesterPublicKey** | **[Number]** | The public key of the Requester node that created this job This can be used to encrypt messages back to the creator | [optional] 
**spec** | [**ModelSpec**](ModelSpec.md) |  | [optional] 
