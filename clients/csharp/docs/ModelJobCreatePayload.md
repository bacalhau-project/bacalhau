# bacalhau-client.Model.ModelJobCreatePayload
## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**ClientID** | **string** | the id of the client that is submitting the job | 
**Context** | **string** | Optional base64-encoded tar file that will be pinned to IPFS and mounted as storage for the job. Not part of the spec so we don&#x27;t flood the transport layer with it (potentially very large). | [optional] 
**Job** | [**ModelJob**](ModelJob.md) |  | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

