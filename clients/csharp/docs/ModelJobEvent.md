# bacalhau-client.Model.ModelJobEvent
## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**APIVersion** | **string** | APIVersion of the Job | [optional] 
**ClientID** | **string** | optional clientID if this is an externally triggered event (like create job) | [optional] 
**Deal** | [**ModelDeal**](ModelDeal.md) |  | [optional] 
**EventName** | **int?** |  | [optional] 
**EventTime** | **string** |  | [optional] 
**JobExecutionPlan** | [**ModelJobExecutionPlan**](ModelJobExecutionPlan.md) |  | [optional] 
**JobID** | **string** |  | [optional] 
**PublishedResult** | [**ModelStorageSpec**](ModelStorageSpec.md) |  | [optional] 
**RunOutput** | [**ModelRunCommandResult**](ModelRunCommandResult.md) |  | [optional] 
**SenderPublicKey** | **List&lt;int?&gt;** |  | [optional] 
**ShardIndex** | **int?** | what shard is this event for | [optional] 
**SourceNodeID** | **string** | the node that emitted this event | [optional] 
**Spec** | [**ModelSpec**](ModelSpec.md) |  | [optional] 
**Status** | **string** |  | [optional] 
**TargetNodeID** | **string** | the node that this event is for e.g. \&quot;AcceptJobBid\&quot; was emitted by Requester but it targeting compute node | [optional] 
**VerificationProposal** | **List&lt;int?&gt;** |  | [optional] 
**VerificationResult** | [**ModelVerificationResult**](ModelVerificationResult.md) |  | [optional] 

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)

