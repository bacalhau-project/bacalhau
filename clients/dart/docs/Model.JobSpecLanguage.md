# swagger.model.ModelJobSpecLanguage

## Load the model package
```dart
import 'package:swagger/api.dart';
```

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**command** | **String** | optional program specified on commandline, like python -c \&quot;print(1+1)\&quot; | [optional] [default to null]
**deterministicExecution** | **bool** | must this job be run in a deterministic context? | [optional] [default to null]
**jobContext** | [**ModelStorageSpec**](ModelStorageSpec.md) |  | [optional] [default to null]
**language** | **String** | e.g. python | [optional] [default to null]
**languageVersion** | **String** | e.g. 3.8 | [optional] [default to null]
**programPath** | **String** | optional program path relative to the context dir. one of Command or ProgramPath must be specified | [optional] [default to null]
**requirementsPath** | **String** | optional requirements.txt (or equivalent) path relative to the context dir | [optional] [default to null]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

