# ModelJob

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**aPIVersion** | [**kotlin.String**](.md) |  |  [optional]
**clientID** | [**kotlin.String**](.md) | The ID of the client that created this job. |  [optional]
**createdAt** | [**kotlin.String**](.md) | Time the job was submitted to the bacalhau network. |  [optional]
**deal** | [**ModelDeal**](ModelDeal.md) |  |  [optional]
**executionPlan** | [**ModelJobExecutionPlan**](ModelJobExecutionPlan.md) |  |  [optional]
**iD** | [**kotlin.String**](.md) | The unique global ID of this job in the bacalhau network. |  [optional]
**jobEvents** | [**kotlin.Array&lt;ModelJobEvent&gt;**](ModelJobEvent.md) | All events associated with the job |  [optional]
**jobState** | [**ModelJobState**](ModelJobState.md) |  |  [optional]
**localJobEvents** | [**kotlin.Array&lt;ModelJobLocalEvent&gt;**](ModelJobLocalEvent.md) | All local events associated with the job |  [optional]
**requesterNodeID** | [**kotlin.String**](.md) | The ID of the requester node that owns this job. |  [optional]
**requesterPublicKey** | [**kotlin.Array&lt;kotlin.Int&gt;**](.md) | The public key of the Requester node that created this job This can be used to encrypt messages back to the creator |  [optional]
**spec** | [**ModelSpec**](ModelSpec.md) |  |  [optional]
