package io.swagger.server.model


/**
 * @param CapacityRequirements 
 * @param ShardID 
 * @param State 
 */
case class Computenode.ActiveJob (
  CapacityRequirements: Option[model.ResourceUsageData],
  ShardID: Option[String],
  State: Option[String]
)

