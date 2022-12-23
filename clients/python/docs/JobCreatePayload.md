# JobCreatePayload

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**api_version** | **str** |  | 
**client_id** | **str** | the id of the client that is submitting the job | 
**context** | **str** | Optional base64-encoded tar file that will be pinned to IPFS and mounted as storage for the job. Not part of the spec so we don&#39;t flood the transport layer with it (potentially very large). | [optional] 
**spec** | **list[int]** | The specification of this job. | 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


