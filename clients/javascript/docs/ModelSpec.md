# BacalhauClient.ModelSpec

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**annotations** | **[String]** | Annotations on the job - could be user or machine assigned | [optional] 
**contexts** | [**[ModelStorageSpec]**](ModelStorageSpec.md) | Input volumes that will not be sharded for example to upload code into a base image every shard will get the full range of context volumes | [optional] 
**doNotTrack** | **Boolean** | Do not track specified by the client | [optional] 
**docker** | [**ModelJobSpecDocker**](ModelJobSpecDocker.md) |  | [optional] 
**engine** | **Number** | e.g. docker or language | [optional] 
**language** | [**ModelJobSpecLanguage**](ModelJobSpecLanguage.md) |  | [optional] 
**publisher** | **Number** | there can be multiple publishers for the job | [optional] 
**resources** | [**ModelResourceUsageConfig**](ModelResourceUsageConfig.md) |  | [optional] 
**sharding** | [**ModelJobShardingConfig**](ModelJobShardingConfig.md) |  | [optional] 
**timeout** | **Number** | How long a job can run in seconds before it is killed. This includes the time required to run, verify and publish results | [optional] 
**verifier** | **Number** |  | [optional] 
**wasm** | [**ModelJobSpecWasm**](ModelJobSpecWasm.md) |  | [optional] 
**inputs** | [**[ModelStorageSpec]**](ModelStorageSpec.md) | the data volumes we will read in the job for example \&quot;read this ipfs cid\&quot; TODO: #667 Replace with \&quot;Inputs\&quot;, \&quot;Outputs\&quot; (note the caps) for yaml/json when we update the n.js file | [optional] 
**outputs** | [**[ModelStorageSpec]**](ModelStorageSpec.md) | the data volumes we will write in the job for example \&quot;write the results to ipfs\&quot; | [optional] 
