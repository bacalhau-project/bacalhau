# swagger.model.ModelDeal

## Load the model package
```dart
import 'package:swagger/api.dart';
```

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**concurrency** | **int** | The maximum number of concurrent compute node bids that will be accepted by the requester node on behalf of the client. | [optional] [default to null]
**confidence** | **int** | The number of nodes that must agree on a verification result this is used by the different verifiers - for example the deterministic verifier requires the winning group size to be at least this size | [optional] [default to null]
**minBids** | **int** | The minimum number of bids that must be received before the Requester node will randomly accept concurrency-many of them. This allows the Requester node to get some level of guarantee that the execution of the jobs will be spread evenly across the network (assuming that this value is some large proportion of the size of the network). | [optional] [default to null]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

