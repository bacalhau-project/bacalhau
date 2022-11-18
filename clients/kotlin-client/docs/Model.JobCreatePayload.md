# ModelJobCreatePayload

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**clientID** | [**kotlin.String**](.md) | the id of the client that is submitting the job | 
**context** | [**kotlin.String**](.md) | Optional base64-encoded tar file that will be pinned to IPFS and mounted as storage for the job. Not part of the spec so we don&#x27;t flood the transport layer with it (potentially very large). |  [optional]
**job** | [**ModelJob**](ModelJob.md) |  | 
