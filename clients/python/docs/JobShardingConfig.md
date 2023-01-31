# JobShardingConfig

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**batch_size** | **int** | how many \&quot;items\&quot; are to be processed in each shard we first apply the glob pattern which will result in a flat list of items this number decides how to group that flat list into actual shards run by compute nodes | [optional] 
**glob_pattern** | **str** | divide the inputs up into the smallest possible unit for example /* would mean \&quot;all top level files or folders\&quot; this being an empty string means \&quot;no sharding\&quot; | [optional] 
**glob_pattern_base_path** | **str** | when using multiple input volumes what path do we treat as the common mount path to apply the glob pattern to | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


