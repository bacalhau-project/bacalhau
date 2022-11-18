# SwaggerClient::ModelJobSpecWasm

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**entry_point** | **String** | The name of the function in the EntryModule to call to run the job. For WASI jobs, this will always be &#x60;_start&#x60;, but jobs can choose to call other WASM functions instead. The EntryPoint must be a zero-parameter zero-result function. | [optional] 
**environment_variables** | **Hash&lt;String, String&gt;** | The variables available in the environment of the running program. | [optional] 
**import_modules** | [**Array&lt;ModelStorageSpec&gt;**](ModelStorageSpec.md) | TODO #880: Other WASM modules whose exports will be available as imports to the EntryModule. | [optional] 
**parameters** | **Array&lt;String&gt;** | The arguments supplied to the program (i.e. as ARGV). | [optional] 

