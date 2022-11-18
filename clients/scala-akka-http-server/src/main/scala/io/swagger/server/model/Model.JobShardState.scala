package io.swagger.server.model


/**
 * @param NodeId which node is running this shard
 * @param PublishedResults 
 * @param RunOutput 
 * @param ShardIndex what shard is this we are running
 * @param State what is the state of the shard on this node
 * @param Status an arbitrary status message
 * @param VerificationProposal the proposed results for this shard this will be resolved by the verifier somehow
 * @param VerificationResult 
 */
case class Model.JobShardState (
  NodeId: Option[String],
  PublishedResults: Option[model.StorageSpec],
  RunOutput: Option[model.RunCommandResult],
  ShardIndex: Option[Int],
  State: Option[Int],
  Status: Option[String],
  VerificationProposal: Option[List[Int]],
  VerificationResult: Option[model.VerificationResult]
)

