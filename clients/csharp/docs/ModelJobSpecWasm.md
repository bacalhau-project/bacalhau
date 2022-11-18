# bacalhau-client.Model.ModelJobSpecWasm
## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**EntryPoint** | **string** | The name of the function in the EntryModule to call to run the job. For WASI jobs, this will always be &#x60;_start&#x60;, but jobs can choose to call other WASM functions instead. The EntryPoint must be a zero-parameter zero-result function. | [optional] 
**EnvironmentVariables** | **Dictionary&lt;string, string&gt;** | The variables available in the environment of the running program. | [optional] 
**ImportModules** | [**List&lt;ModelStorageSpec&gt;**](ModelStorageSpec.md) | TODO #880: Other WASM modules whose exports will be available as imports to the EntryModule. | [optional] 
**Parameters** | **List&lt;string&gt;** | The arguments supplied to the program (i.e. as ARGV). | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

