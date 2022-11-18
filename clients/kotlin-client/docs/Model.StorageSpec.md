# ModelStorageSpec

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**cID** | [**kotlin.String**](.md) | The unique ID of the data, where it makes sense (for example, in an IPFS storage spec this will be the data&#x27;s CID). NOTE: The below is capitalized to match IPFS &amp; IPLD (even though it&#x27;s out of golang fmt) |  [optional]
**metadata** | [**kotlin.collections.Map&lt;kotlin.String, kotlin.String&gt;**](.md) | Additional properties specific to each driver |  [optional]
**name** | [**kotlin.String**](.md) | Name of the spec&#x27;s data, for reference. |  [optional]
**storageSource** | [**kotlin.Int**](.md) | StorageSource is the abstract source of the data. E.g. a storage source might be a URL download, but doesn&#x27;t specify how the execution engine does the download or what it will do with the downloaded data. |  [optional]
**uRL** | [**kotlin.String**](.md) | Source URL of the data |  [optional]
**path** | [**kotlin.String**](.md) | The path that the spec&#x27;s data should be mounted on, where it makes sense (for example, in a Docker storage spec this will be a filesystem path). TODO: #668 Replace with \&quot;Path\&quot; (note the caps) for yaml/json when we update the n.js file |  [optional]
