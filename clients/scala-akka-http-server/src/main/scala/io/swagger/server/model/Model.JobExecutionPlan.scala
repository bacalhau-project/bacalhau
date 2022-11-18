package io.swagger.server.model


/**
 * @param ShardsTotal how many shards are there in total for this job we are expecting this number x concurrency total JobShardState objects for this job
 */
case class Model.JobExecutionPlan (
  ShardsTotal: Option[Int]
)

