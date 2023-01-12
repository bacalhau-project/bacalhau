# JobSpecWasm

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**entry_module** | [**JobSpecWasmEntryModule**](JobSpecWasmEntryModule.md) |  | [optional] 
**entry_point** | **str** | The name of the function in the EntryModule to call to run the job. For WASI jobs, this will always be &#x60;_start&#x60;, but jobs can choose to call other WASM functions instead. The EntryPoint must be a zero-parameter zero-result function. | [optional] 
**environment_variables** | **dict(str, str)** | The variables available in the environment of the running program. | [optional] 
**import_modules** | [**list[StorageSpec]**](StorageSpec.md) | TODO #880: Other WASM modules whose exports will be available as imports to the EntryModule. | [optional] 
**parameters** | **list[str]** | The arguments supplied to the program (i.e. as ARGV). | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


