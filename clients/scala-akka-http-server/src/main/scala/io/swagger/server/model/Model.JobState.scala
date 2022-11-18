package io.swagger.server.model


/**
 * @param Nodes 
 */
case class Model.JobState (
  Nodes: Option[Map[String, model.JobNodeState]]
)

