package io.swagger.server.model


/**
 * @param BiddingNodesCount 
 * @param CompletedNodesCount 
 * @param ShardID 
 * @param State 
 */
case class Requesternode.ActiveJob (
  BiddingNodesCount: Option[Int],
  CompletedNodesCount: Option[Int],
  ShardID: Option[String],
  State: Option[String]
)

