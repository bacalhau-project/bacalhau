# ModelJobShardState

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**node_id** | **string** | which node is running this shard | [optional] 
**published_results** | [**\Swagger\Client\Model\ModelStorageSpec**](ModelStorageSpec.md) |  | [optional] 
**run_output** | [**\Swagger\Client\Model\ModelRunCommandResult**](ModelRunCommandResult.md) |  | [optional] 
**shard_index** | **int** | what shard is this we are running | [optional] 
**state** | **int** | what is the state of the shard on this node | [optional] 
**status** | **string** | an arbitrary status message | [optional] 
**verification_proposal** | **int[]** | the proposed results for this shard this will be resolved by the verifier somehow | [optional] 
**verification_result** | [**\Swagger\Client\Model\ModelVerificationResult**](ModelVerificationResult.md) |  | [optional] 

[[Back to Model list]](../../README.md#documentation-for-models) [[Back to API list]](../../README.md#documentation-for-api-endpoints) [[Back to README]](../../README.md)

