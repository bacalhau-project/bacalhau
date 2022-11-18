# swagger.model.ModelJobEvent

## Load the model package
```dart
import 'package:swagger/api.dart';
```

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**aPIVersion** | **String** | APIVersion of the Job | [optional] [default to null]
**clientID** | **String** | optional clientID if this is an externally triggered event (like create job) | [optional] [default to null]
**deal** | [**ModelDeal**](ModelDeal.md) |  | [optional] [default to null]
**eventName** | **int** |  | [optional] [default to null]
**eventTime** | **String** |  | [optional] [default to null]
**jobExecutionPlan** | [**ModelJobExecutionPlan**](ModelJobExecutionPlan.md) |  | [optional] [default to null]
**jobID** | **String** |  | [optional] [default to null]
**publishedResult** | [**ModelStorageSpec**](ModelStorageSpec.md) |  | [optional] [default to null]
**runOutput** | [**ModelRunCommandResult**](ModelRunCommandResult.md) |  | [optional] [default to null]
**senderPublicKey** | **List&lt;int&gt;** |  | [optional] [default to []]
**shardIndex** | **int** | what shard is this event for | [optional] [default to null]
**sourceNodeID** | **String** | the node that emitted this event | [optional] [default to null]
**spec** | [**ModelSpec**](ModelSpec.md) |  | [optional] [default to null]
**status** | **String** |  | [optional] [default to null]
**targetNodeID** | **String** | the node that this event is for e.g. \&quot;AcceptJobBid\&quot; was emitted by Requester but it targeting compute node | [optional] [default to null]
**verificationProposal** | **List&lt;int&gt;** |  | [optional] [default to []]
**verificationResult** | [**ModelVerificationResult**](ModelVerificationResult.md) |  | [optional] [default to null]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

