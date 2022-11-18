package io.swagger.server.model


/**
 * @param APIVersion APIVersion of the Job for example: ''V1beta1''
 * @param ClientID optional clientID if this is an externally triggered event (like create job) for example: ''ac13188e93c97a9c2e7cf8e86c7313156a73436036f30da1ececc2ce79f9ea51''
 * @param Deal 
 * @param EventName 
 * @param EventTime  for example: ''2022-11-17T13:32:55.756658941Z''
 * @param JobExecutionPlan 
 * @param JobID  for example: ''9304c616-291f-41ad-b862-54e133c0149e''
 * @param PublishedResult 
 * @param RunOutput 
 * @param SenderPublicKey 
 * @param ShardIndex what shard is this event for
 * @param SourceNodeID the node that emitted this event for example: ''QmXaXu9N5GNetatsvwnTfQqNtSeKAD6uCmarbh3LMRYAcF''
 * @param Spec 
 * @param Status  for example: ''Got results proposal of length: 0''
 * @param TargetNodeID the node that this event is for e.g. \"AcceptJobBid\" was emitted by Requester but it targeting compute node for example: ''QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL''
 * @param VerificationProposal 
 * @param VerificationResult 
 */
case class Model.JobEvent (
  APIVersion: Option[String],
  ClientID: Option[String],
  Deal: Option[model.Deal],
  EventName: Option[Int],
  EventTime: Option[String],
  JobExecutionPlan: Option[model.JobExecutionPlan],
  JobID: Option[String],
  PublishedResult: Option[model.StorageSpec],
  RunOutput: Option[model.RunCommandResult],
  SenderPublicKey: Option[List[Int]],
  ShardIndex: Option[Int],
  SourceNodeID: Option[String],
  Spec: Option[model.Spec],
  Status: Option[String],
  TargetNodeID: Option[String],
  VerificationProposal: Option[List[Int]],
  VerificationResult: Option[model.VerificationResult]
)

