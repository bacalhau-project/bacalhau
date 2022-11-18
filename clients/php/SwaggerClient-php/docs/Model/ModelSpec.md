# ModelSpec

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**annotations** | **string[]** | Annotations on the job - could be user or machine assigned | [optional] 
**contexts** | [**\Swagger\Client\Model\ModelStorageSpec[]**](ModelStorageSpec.md) | Input volumes that will not be sharded for example to upload code into a base image every shard will get the full range of context volumes | [optional] 
**do_not_track** | **bool** | Do not track specified by the client | [optional] 
**docker** | [**\Swagger\Client\Model\ModelJobSpecDocker**](ModelJobSpecDocker.md) |  | [optional] 
**engine** | **int** | e.g. docker or language | [optional] 
**language** | [**\Swagger\Client\Model\ModelJobSpecLanguage**](ModelJobSpecLanguage.md) |  | [optional] 
**publisher** | **int** | there can be multiple publishers for the job | [optional] 
**resources** | [**\Swagger\Client\Model\ModelResourceUsageConfig**](ModelResourceUsageConfig.md) |  | [optional] 
**sharding** | [**\Swagger\Client\Model\ModelJobShardingConfig**](ModelJobShardingConfig.md) |  | [optional] 
**timeout** | **float** | How long a job can run in seconds before it is killed. This includes the time required to run, verify and publish results | [optional] 
**verifier** | **int** |  | [optional] 
**wasm** | [**\Swagger\Client\Model\ModelJobSpecWasm**](ModelJobSpecWasm.md) |  | [optional] 
**inputs** | [**\Swagger\Client\Model\ModelStorageSpec[]**](ModelStorageSpec.md) | the data volumes we will read in the job for example \&quot;read this ipfs cid\&quot; TODO: #667 Replace with \&quot;Inputs\&quot;, \&quot;Outputs\&quot; (note the caps) for yaml/json when we update the n.js file | [optional] 
**outputs** | [**\Swagger\Client\Model\ModelStorageSpec[]**](ModelStorageSpec.md) | the data volumes we will write in the job for example \&quot;write the results to ipfs\&quot; | [optional] 

[[Back to Model list]](../../README.md#documentation-for-models) [[Back to API list]](../../README.md#documentation-for-api-endpoints) [[Back to README]](../../README.md)

