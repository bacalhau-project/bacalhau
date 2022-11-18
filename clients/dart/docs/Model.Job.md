# swagger.model.ModelJob

## Load the model package
```dart
import 'package:swagger/api.dart';
```

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**aPIVersion** | **String** |  | [optional] [default to null]
**clientID** | **String** | The ID of the client that created this job. | [optional] [default to null]
**createdAt** | **String** | Time the job was submitted to the bacalhau network. | [optional] [default to null]
**deal** | [**ModelDeal**](ModelDeal.md) |  | [optional] [default to null]
**executionPlan** | [**ModelJobExecutionPlan**](ModelJobExecutionPlan.md) |  | [optional] [default to null]
**ID** | **String** | The unique global ID of this job in the bacalhau network. | [optional] [default to null]
**jobEvents** | [**List&lt;ModelJobEvent&gt;**](ModelJobEvent.md) | All events associated with the job | [optional] [default to []]
**jobState** | [**ModelJobState**](ModelJobState.md) |  | [optional] [default to null]
**localJobEvents** | [**List&lt;ModelJobLocalEvent&gt;**](ModelJobLocalEvent.md) | All local events associated with the job | [optional] [default to []]
**requesterNodeID** | **String** | The ID of the requester node that owns this job. | [optional] [default to null]
**requesterPublicKey** | **List&lt;int&gt;** | The public key of the Requester node that created this job This can be used to encrypt messages back to the creator | [optional] [default to []]
**spec** | [**ModelSpec**](ModelSpec.md) |  | [optional] [default to null]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

