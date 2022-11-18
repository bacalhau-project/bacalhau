# swagger.model.ModelJobShardingConfig

## Load the model package
```dart
import 'package:swagger/api.dart';
```

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**batchSize** | **int** | how many \&quot;items\&quot; are to be processed in each shard we first apply the glob pattern which will result in a flat list of items this number decides how to group that flat list into actual shards run by compute nodes | [optional] [default to null]
**globPattern** | **String** | divide the inputs up into the smallest possible unit for example /_* would mean \&quot;all top level files or folders\&quot; this being an empty string means \&quot;no sharding\&quot; | [optional] [default to null]
**globPatternBasePath** | **String** | when using multiple input volumes what path do we treat as the common mount path to apply the glob pattern to | [optional] [default to null]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

