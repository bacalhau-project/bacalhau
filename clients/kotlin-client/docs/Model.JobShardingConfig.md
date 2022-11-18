# ModelJobShardingConfig

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**batchSize** | [**kotlin.Int**](.md) | how many \&quot;items\&quot; are to be processed in each shard we first apply the glob pattern which will result in a flat list of items this number decides how to group that flat list into actual shards run by compute nodes |  [optional]
**globPattern** | [**kotlin.String**](.md) | divide the inputs up into the smallest possible unit for example /_* would mean \&quot;all top level files or folders\&quot; this being an empty string means \&quot;no sharding\&quot; |  [optional]
**globPatternBasePath** | [**kotlin.String**](.md) | when using multiple input volumes what path do we treat as the common mount path to apply the glob pattern to |  [optional]
