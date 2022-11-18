# SwaggerClient::ModelJobShardState

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**node_id** | **String** | which node is running this shard | [optional] 
**published_results** | [**ModelStorageSpec**](ModelStorageSpec.md) |  | [optional] 
**run_output** | [**ModelRunCommandResult**](ModelRunCommandResult.md) |  | [optional] 
**shard_index** | **Integer** | what shard is this we are running | [optional] 
**state** | **Integer** | what is the state of the shard on this node | [optional] 
**status** | **String** | an arbitrary status message | [optional] 
**verification_proposal** | **Array&lt;Integer&gt;** | the proposed results for this shard this will be resolved by the verifier somehow | [optional] 
**verification_result** | [**ModelVerificationResult**](ModelVerificationResult.md) |  | [optional] 

