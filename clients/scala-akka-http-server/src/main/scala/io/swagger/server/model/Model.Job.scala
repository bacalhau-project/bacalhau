package io.swagger.server.model


/**
 * @param APIVersion  for example: ''V1beta1''
 * @param ClientID The ID of the client that created this job. for example: ''ac13188e93c97a9c2e7cf8e86c7313156a73436036f30da1ececc2ce79f9ea51''
 * @param CreatedAt Time the job was submitted to the bacalhau network. for example: ''2022-11-17T13:29:01.871140291Z''
 * @param Deal 
 * @param ExecutionPlan 
 * @param ID The unique global ID of this job in the bacalhau network. for example: ''92d5d4ee-3765-4f78-8353-623f5f26df08''
 * @param JobEvents All events associated with the job
 * @param JobState 
 * @param LocalJobEvents All local events associated with the job
 * @param RequesterNodeID The ID of the requester node that owns this job. for example: ''QmXaXu9N5GNetatsvwnTfQqNtSeKAD6uCmarbh3LMRYAcF''
 * @param RequesterPublicKey The public key of the Requester node that created this job This can be used to encrypt messages back to the creator
 * @param Spec 
 */
case class Model.Job (
  APIVersion: Option[String],
  ClientID: Option[String],
  CreatedAt: Option[String],
  Deal: Option[model.Deal],
  ExecutionPlan: Option[model.JobExecutionPlan],
  ID: Option[String],
  JobEvents: Option[List[model.JobEvent]],
  JobState: Option[model.JobState],
  LocalJobEvents: Option[List[model.JobLocalEvent]],
  RequesterNodeID: Option[String],
  RequesterPublicKey: Option[List[Int]],
  Spec: Option[model.Spec]
)

