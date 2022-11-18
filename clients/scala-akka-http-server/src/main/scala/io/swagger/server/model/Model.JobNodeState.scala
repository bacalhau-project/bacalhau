package io.swagger.server.model


/**
 * @param Shards 
 */
case class Model.JobNodeState (
  Shards: Option[Map[String, model.JobShardState]]
)

