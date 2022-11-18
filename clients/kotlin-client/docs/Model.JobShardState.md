# ModelJobShardState

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**nodeId** | [**kotlin.String**](.md) | which node is running this shard |  [optional]
**publishedResults** | [**ModelStorageSpec**](ModelStorageSpec.md) |  |  [optional]
**runOutput** | [**ModelRunCommandResult**](ModelRunCommandResult.md) |  |  [optional]
**shardIndex** | [**kotlin.Int**](.md) | what shard is this we are running |  [optional]
**state** | [**kotlin.Int**](.md) | what is the state of the shard on this node |  [optional]
**status** | [**kotlin.String**](.md) | an arbitrary status message |  [optional]
**verificationProposal** | [**kotlin.Array&lt;kotlin.Int&gt;**](.md) | the proposed results for this shard this will be resolved by the verifier somehow |  [optional]
**verificationResult** | [**ModelVerificationResult**](ModelVerificationResult.md) |  |  [optional]
