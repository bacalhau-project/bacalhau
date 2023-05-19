# StorageSpec

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**cid** | **str** | The unique ID of the data, where it makes sense (for example, in an IPFS storage spec this will be the data&#x27;s CID). NOTE: The below is capitalized to match IPFS &amp; IPLD (even though it&#x27;s out of golang fmt) | [optional]
**metadata** | **dict(str, str)** | Additional properties specific to each driver | [optional]
**name** | **str** | Name of the spec&#x27;s data, for reference. | [optional]
**source_path** | **str** | The path of the host data if we are using local directory paths | [optional]
**storage_source** | **AllOfStorageSpecStorageSource** | StorageSource is the abstract source of the data. E.g. a storage source might be a URL download, but doesn&#x27;t specify how the execution engine does the download or what it will do with the downloaded data. | [optional]
**url** | **str** | Source URL of the data | [optional]
**path** | **str** | The path that the spec&#x27;s data should be mounted on, where it makes sense (for example, in a Docker storage spec this will be a filesystem path). TODO: #668 Replace with \&quot;Path\&quot; (note the caps) for yaml/json when we update the n.js file | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
