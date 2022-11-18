# swagger.model.ModelJobShardState

## Load the model package
```dart
import 'package:swagger/api.dart';
```

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**nodeId** | **String** | which node is running this shard | [optional] [default to null]
**publishedResults** | [**ModelStorageSpec**](ModelStorageSpec.md) |  | [optional] [default to null]
**runOutput** | [**ModelRunCommandResult**](ModelRunCommandResult.md) |  | [optional] [default to null]
**shardIndex** | **int** | what shard is this we are running | [optional] [default to null]
**state** | **int** | what is the state of the shard on this node | [optional] [default to null]
**status** | **String** | an arbitrary status message | [optional] [default to null]
**verificationProposal** | **List&lt;int&gt;** | the proposed results for this shard this will be resolved by the verifier somehow | [optional] [default to []]
**verificationResult** | [**ModelVerificationResult**](ModelVerificationResult.md) |  | [optional] [default to null]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

