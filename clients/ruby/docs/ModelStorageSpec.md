# SwaggerClient::ModelStorageSpec

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**cid** | **String** | The unique ID of the data, where it makes sense (for example, in an IPFS storage spec this will be the data&#x27;s CID). NOTE: The below is capitalized to match IPFS &amp; IPLD (even though it&#x27;s out of golang fmt) | [optional] 
**metadata** | **Hash&lt;String, String&gt;** | Additional properties specific to each driver | [optional] 
**name** | **String** | Name of the spec&#x27;s data, for reference. | [optional] 
**storage_source** | **Integer** | StorageSource is the abstract source of the data. E.g. a storage source might be a URL download, but doesn&#x27;t specify how the execution engine does the download or what it will do with the downloaded data. | [optional] 
**url** | **String** | Source URL of the data | [optional] 
**path** | **String** | The path that the spec&#x27;s data should be mounted on, where it makes sense (for example, in a Docker storage spec this will be a filesystem path). TODO: #668 Replace with \&quot;Path\&quot; (note the caps) for yaml/json when we update the n.js file | [optional] 

