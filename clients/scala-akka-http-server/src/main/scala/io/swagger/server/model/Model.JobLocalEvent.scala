package io.swagger.server.model


/**
 * @param EventName 
 * @param JobID 
 * @param ShardIndex 
 * @param TargetNodeID 
 */
case class Model.JobLocalEvent (
  EventName: Option[Int],
  JobID: Option[String],
  ShardIndex: Option[Int],
  TargetNodeID: Option[String]
)

