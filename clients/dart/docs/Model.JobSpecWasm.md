# swagger.model.ModelJobSpecWasm

## Load the model package
```dart
import 'package:swagger/api.dart';
```

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**entryPoint** | **String** | The name of the function in the EntryModule to call to run the job. For WASI jobs, this will always be &#x60;_start&#x60;, but jobs can choose to call other WASM functions instead. The EntryPoint must be a zero-parameter zero-result function. | [optional] [default to null]
**environmentVariables** | **Map&lt;String, String&gt;** | The variables available in the environment of the running program. | [optional] [default to {}]
**importModules** | [**List&lt;ModelStorageSpec&gt;**](ModelStorageSpec.md) | TODO #880: Other WASM modules whose exports will be available as imports to the EntryModule. | [optional] [default to []]
**parameters** | **List&lt;String&gt;** | The arguments supplied to the program (i.e. as ARGV). | [optional] [default to []]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

