# ModelJobShardState

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**node_id** | **str** | which node is running this shard | [optional] 
**published_results** | [**ModelStorageSpec**](ModelStorageSpec.md) |  | [optional] 
**run_output** | [**ModelRunCommandResult**](ModelRunCommandResult.md) |  | [optional] 
**shard_index** | **int** | what shard is this we are running | [optional] 
**state** | **int** | what is the state of the shard on this node | [optional] 
**status** | **str** | an arbitrary status message | [optional] 
**verification_proposal** | **list[int]** | the proposed results for this shard this will be resolved by the verifier somehow | [optional] 
**verification_result** | [**ModelVerificationResult**](ModelVerificationResult.md) |  | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

