# ModelJobEvent

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**aPIVersion** | [**kotlin.String**](.md) | APIVersion of the Job |  [optional]
**clientID** | [**kotlin.String**](.md) | optional clientID if this is an externally triggered event (like create job) |  [optional]
**deal** | [**ModelDeal**](ModelDeal.md) |  |  [optional]
**eventName** | [**kotlin.Int**](.md) |  |  [optional]
**eventTime** | [**kotlin.String**](.md) |  |  [optional]
**jobExecutionPlan** | [**ModelJobExecutionPlan**](ModelJobExecutionPlan.md) |  |  [optional]
**jobID** | [**kotlin.String**](.md) |  |  [optional]
**publishedResult** | [**ModelStorageSpec**](ModelStorageSpec.md) |  |  [optional]
**runOutput** | [**ModelRunCommandResult**](ModelRunCommandResult.md) |  |  [optional]
**senderPublicKey** | [**kotlin.Array&lt;kotlin.Int&gt;**](.md) |  |  [optional]
**shardIndex** | [**kotlin.Int**](.md) | what shard is this event for |  [optional]
**sourceNodeID** | [**kotlin.String**](.md) | the node that emitted this event |  [optional]
**spec** | [**ModelSpec**](ModelSpec.md) |  |  [optional]
**status** | [**kotlin.String**](.md) |  |  [optional]
**targetNodeID** | [**kotlin.String**](.md) | the node that this event is for e.g. \&quot;AcceptJobBid\&quot; was emitted by Requester but it targeting compute node |  [optional]
**verificationProposal** | [**kotlin.Array&lt;kotlin.Int&gt;**](.md) |  |  [optional]
**verificationResult** | [**ModelVerificationResult**](ModelVerificationResult.md) |  |  [optional]
