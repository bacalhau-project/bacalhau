# ModelDeal

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**concurrency** | [**kotlin.Int**](.md) | The maximum number of concurrent compute node bids that will be accepted by the requester node on behalf of the client. |  [optional]
**confidence** | [**kotlin.Int**](.md) | The number of nodes that must agree on a verification result this is used by the different verifiers - for example the deterministic verifier requires the winning group size to be at least this size |  [optional]
**minBids** | [**kotlin.Int**](.md) | The minimum number of bids that must be received before the Requester node will randomly accept concurrency-many of them. This allows the Requester node to get some level of guarantee that the execution of the jobs will be spread evenly across the network (assuming that this value is some large proportion of the size of the network). |  [optional]
