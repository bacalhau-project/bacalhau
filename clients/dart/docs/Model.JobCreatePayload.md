# swagger.model.ModelJobCreatePayload

## Load the model package
```dart
import 'package:swagger/api.dart';
```

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**clientID** | **String** | the id of the client that is submitting the job | [default to null]
**context** | **String** | Optional base64-encoded tar file that will be pinned to IPFS and mounted as storage for the job. Not part of the spec so we don&#x27;t flood the transport layer with it (potentially very large). | [optional] [default to null]
**job** | [**ModelJob**](ModelJob.md) |  | [default to null]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

