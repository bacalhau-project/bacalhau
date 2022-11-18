# ModelJobSpecWasm

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**entryPoint** | [**kotlin.String**](.md) | The name of the function in the EntryModule to call to run the job. For WASI jobs, this will always be &#x60;_start&#x60;, but jobs can choose to call other WASM functions instead. The EntryPoint must be a zero-parameter zero-result function. |  [optional]
**environmentVariables** | [**kotlin.collections.Map&lt;kotlin.String, kotlin.String&gt;**](.md) | The variables available in the environment of the running program. |  [optional]
**importModules** | [**kotlin.Array&lt;ModelStorageSpec&gt;**](ModelStorageSpec.md) | TODO #880: Other WASM modules whose exports will be available as imports to the EntryModule. |  [optional]
**parameters** | [**kotlin.Array&lt;kotlin.String&gt;**](.md) | The arguments supplied to the program (i.e. as ARGV). |  [optional]
