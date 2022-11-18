package io.swagger.server.model


/**
 * @param Data 
 * @param NodeID 
 * @param ShardIndex 
 */
case class Model.PublishedResult (
  Data: Option[model.StorageSpec],
  NodeID: Option[String],
  ShardIndex: Option[Int]
)

