package io.swagger.server.model


/**
 * @param Concurrency The maximum number of concurrent compute node bids that will be accepted by the requester node on behalf of the client.
 * @param Confidence The number of nodes that must agree on a verification result this is used by the different verifiers - for example the deterministic verifier requires the winning group size to be at least this size
 * @param MinBids The minimum number of bids that must be received before the Requester node will randomly accept concurrency-many of them. This allows the Requester node to get some level of guarantee that the execution of the jobs will be spread evenly across the network (assuming that this value is some large proportion of the size of the network).
 */
case class Model.Deal (
  Concurrency: Option[Int],
  Confidence: Option[Int],
  MinBids: Option[Int]
)

