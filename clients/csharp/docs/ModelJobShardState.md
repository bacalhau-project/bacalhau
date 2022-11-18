# bacalhau-client.Model.ModelJobShardState
## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**NodeId** | **string** | which node is running this shard | [optional] 
**PublishedResults** | [**ModelStorageSpec**](ModelStorageSpec.md) |  | [optional] 
**RunOutput** | [**ModelRunCommandResult**](ModelRunCommandResult.md) |  | [optional] 
**ShardIndex** | **int?** | what shard is this we are running | [optional] 
**State** | **int?** | what is the state of the shard on this node | [optional] 
**Status** | **string** | an arbitrary status message | [optional] 
**VerificationProposal** | **List&lt;int?&gt;** | the proposed results for this shard this will be resolved by the verifier somehow | [optional] 
**VerificationResult** | [**ModelVerificationResult**](ModelVerificationResult.md) |  | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

