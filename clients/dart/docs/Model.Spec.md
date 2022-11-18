# swagger.model.ModelSpec

## Load the model package
```dart
import 'package:swagger/api.dart';
```

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**annotations** | **List&lt;String&gt;** | Annotations on the job - could be user or machine assigned | [optional] [default to []]
**contexts** | [**List&lt;ModelStorageSpec&gt;**](ModelStorageSpec.md) | Input volumes that will not be sharded for example to upload code into a base image every shard will get the full range of context volumes | [optional] [default to []]
**doNotTrack** | **bool** | Do not track specified by the client | [optional] [default to null]
**docker** | [**ModelJobSpecDocker**](ModelJobSpecDocker.md) |  | [optional] [default to null]
**engine** | **int** | e.g. docker or language | [optional] [default to null]
**language** | [**ModelJobSpecLanguage**](ModelJobSpecLanguage.md) |  | [optional] [default to null]
**publisher** | **int** | there can be multiple publishers for the job | [optional] [default to null]
**resources** | [**ModelResourceUsageConfig**](ModelResourceUsageConfig.md) |  | [optional] [default to null]
**sharding** | [**ModelJobShardingConfig**](ModelJobShardingConfig.md) |  | [optional] [default to null]
**timeout** | **double** | How long a job can run in seconds before it is killed. This includes the time required to run, verify and publish results | [optional] [default to null]
**verifier** | **int** |  | [optional] [default to null]
**wasm** | [**ModelJobSpecWasm**](ModelJobSpecWasm.md) |  | [optional] [default to null]
**inputs** | [**List&lt;ModelStorageSpec&gt;**](ModelStorageSpec.md) | the data volumes we will read in the job for example \&quot;read this ipfs cid\&quot; TODO: #667 Replace with \&quot;Inputs\&quot;, \&quot;Outputs\&quot; (note the caps) for yaml/json when we update the n.js file | [optional] [default to []]
**outputs** | [**List&lt;ModelStorageSpec&gt;**](ModelStorageSpec.md) | the data volumes we will write in the job for example \&quot;write the results to ipfs\&quot; | [optional] [default to []]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

