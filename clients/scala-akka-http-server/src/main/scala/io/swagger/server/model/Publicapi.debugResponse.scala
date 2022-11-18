package io.swagger.server.model


/**
 * @param AvailableComputeCapacity 
 * @param ComputeJobs 
 * @param RequesterJobs 
 */
case class Publicapi.debugResponse (
  AvailableComputeCapacity: Option[model.ResourceUsageData],
  ComputeJobs: Option[List[computenode.ActiveJob]],
  RequesterJobs: Option[List[requesternode.ActiveJob]]
)

