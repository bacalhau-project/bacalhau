# ModelJobShardState

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**nodeId** | **String** | which node is running this shard |  [optional]
**publishedResults** | [**ModelStorageSpec**](ModelStorageSpec.md) |  |  [optional]
**runOutput** | [**ModelRunCommandResult**](ModelRunCommandResult.md) |  |  [optional]
**shardIndex** | **Integer** | what shard is this we are running |  [optional]
**state** | **Integer** | what is the state of the shard on this node |  [optional]
**status** | **String** | an arbitrary status message |  [optional]
**verificationProposal** | **List&lt;Integer&gt;** | the proposed results for this shard this will be resolved by the verifier somehow |  [optional]
**verificationResult** | [**ModelVerificationResult**](ModelVerificationResult.md) |  |  [optional]
