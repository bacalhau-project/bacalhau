# bacalhau-client.Model.ModelSpec
## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Annotations** | **List&lt;string&gt;** | Annotations on the job - could be user or machine assigned | [optional] 
**Contexts** | [**List&lt;ModelStorageSpec&gt;**](ModelStorageSpec.md) | Input volumes that will not be sharded for example to upload code into a base image every shard will get the full range of context volumes | [optional] 
**DoNotTrack** | **bool?** | Do not track specified by the client | [optional] 
**Docker** | [**ModelJobSpecDocker**](ModelJobSpecDocker.md) |  | [optional] 
**Engine** | **int?** | e.g. docker or language | [optional] 
**Language** | [**ModelJobSpecLanguage**](ModelJobSpecLanguage.md) |  | [optional] 
**Publisher** | **int?** | there can be multiple publishers for the job | [optional] 
**Resources** | [**ModelResourceUsageConfig**](ModelResourceUsageConfig.md) |  | [optional] 
**Sharding** | [**ModelJobShardingConfig**](ModelJobShardingConfig.md) |  | [optional] 
**Timeout** | [**decimal?**](BigDecimal.md) | How long a job can run in seconds before it is killed. This includes the time required to run, verify and publish results | [optional] 
**Verifier** | **int?** |  | [optional] 
**Wasm** | [**ModelJobSpecWasm**](ModelJobSpecWasm.md) |  | [optional] 
**Inputs** | [**List&lt;ModelStorageSpec&gt;**](ModelStorageSpec.md) | the data volumes we will read in the job for example \&quot;read this ipfs cid\&quot; TODO: #667 Replace with \&quot;Inputs\&quot;, \&quot;Outputs\&quot; (note the caps) for yaml/json when we update the n.js file | [optional] 
**Outputs** | [**List&lt;ModelStorageSpec&gt;**](ModelStorageSpec.md) | the data volumes we will write in the job for example \&quot;write the results to ipfs\&quot; | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

