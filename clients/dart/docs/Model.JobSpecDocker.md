# swagger.model.ModelJobSpecDocker

## Load the model package
```dart
import 'package:swagger/api.dart';
```

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**entrypoint** | **List&lt;String&gt;** | optionally override the default entrypoint | [optional] [default to []]
**environmentVariables** | **List&lt;String&gt;** | a map of env to run the container with | [optional] [default to []]
**image** | **String** | this should be pullable by docker | [optional] [default to null]
**workingDirectory** | **String** | working directory inside the container | [optional] [default to null]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

