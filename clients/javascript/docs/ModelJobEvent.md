# BacalhauClient.ModelJobEvent

## Properties
Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**aPIVersion** | **String** | APIVersion of the Job | [optional] 
**clientID** | **String** | optional clientID if this is an externally triggered event (like create job) | [optional] 
**deal** | [**ModelDeal**](ModelDeal.md) |  | [optional] 
**eventName** | **Number** |  | [optional] 
**eventTime** | **String** |  | [optional] 
**jobExecutionPlan** | [**ModelJobExecutionPlan**](ModelJobExecutionPlan.md) |  | [optional] 
**jobID** | **String** |  | [optional] 
**publishedResult** | [**ModelStorageSpec**](ModelStorageSpec.md) |  | [optional] 
**runOutput** | [**ModelRunCommandResult**](ModelRunCommandResult.md) |  | [optional] 
**senderPublicKey** | **[Number]** |  | [optional] 
**shardIndex** | **Number** | what shard is this event for | [optional] 
**sourceNodeID** | **String** | the node that emitted this event | [optional] 
**spec** | [**ModelSpec**](ModelSpec.md) |  | [optional] 
**status** | **String** |  | [optional] 
**targetNodeID** | **String** | the node that this event is for e.g. \&quot;AcceptJobBid\&quot; was emitted by Requester but it targeting compute node | [optional] 
**verificationProposal** | **[Number]** |  | [optional] 
**verificationResult** | [**ModelVerificationResult**](ModelVerificationResult.md) |  | [optional] 
