# Spec

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**annotations** | **list[str]** | Annotations on the job - could be user or machine assigned | [optional]
**deal** | **AllOfSpecDeal** | The deal the client has made, such as which job bids they have accepted. | [optional]
**do_not_track** | **bool** | Do not track specified by the client | [optional]
**docker** | **AllOfSpecDocker** | executor specific data | [optional]
**engine** | **AllOfSpecEngine** | e.g. docker or language | [optional]
**language** | [**JobSpecLanguage**](JobSpecLanguage.md) |  | [optional]
**network** | **AllOfSpecNetwork** | The type of networking access that the job needs | [optional]
**node_selectors** | [**list[LabelSelectorRequirement]**](LabelSelectorRequirement.md) | NodeSelectors is a selector which must be true for the compute node to run this job. | [optional]
**publisher** | **AllOfSpecPublisher** | there can be multiple publishers for the job | [optional]
**resources** | **AllOfSpecResources** | the compute (cpu, ram) resources this job requires | [optional]
**timeout** | **float** | How long a job can run in seconds before it is killed. This includes the time required to run, verify and publish results | [optional]
**verifier** | [**Verifier**](Verifier.md) |  | [optional]
**wasm** | [**JobSpecWasm**](JobSpecWasm.md) |  | [optional]
**inputs** | [**list[StorageSpec]**](StorageSpec.md) | the data volumes we will read in the job for example \&quot;read this ipfs cid\&quot; TODO: #667 Replace with \&quot;Inputs\&quot;, \&quot;Outputs\&quot; (note the caps) for yaml/json when we update the n.js file | [optional]
**outputs** | [**list[StorageSpec]**](StorageSpec.md) | the data volumes we will write in the job for example \&quot;write the results to ipfs\&quot; | [optional]

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)
