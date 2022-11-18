package io.swagger.server.model

import java.math.BigDecimal

/**
 * @param Annotations Annotations on the job - could be user or machine assigned
 * @param Contexts Input volumes that will not be sharded for example to upload code into a base image every shard will get the full range of context volumes
 * @param DoNotTrack Do not track specified by the client
 * @param Docker 
 * @param Engine e.g. docker or language
 * @param Language 
 * @param Publisher there can be multiple publishers for the job
 * @param Resources 
 * @param Sharding 
 * @param Timeout How long a job can run in seconds before it is killed. This includes the time required to run, verify and publish results
 * @param Verifier 
 * @param Wasm 
 * @param inputs the data volumes we will read in the job for example \"read this ipfs cid\" TODO: #667 Replace with \"Inputs\", \"Outputs\" (note the caps) for yaml/json when we update the n.js file
 * @param outputs the data volumes we will write in the job for example \"write the results to ipfs\"
 */
case class Model.Spec (
  Annotations: Option[List[String]],
  Contexts: Option[List[model.StorageSpec]],
  DoNotTrack: Option[Boolean],
  Docker: Option[model.JobSpecDocker],
  Engine: Option[Int],
  Language: Option[model.JobSpecLanguage],
  Publisher: Option[Int],
  Resources: Option[model.ResourceUsageConfig],
  Sharding: Option[model.JobShardingConfig],
  Timeout: Option[BigDecimal],
  Verifier: Option[Int],
  Wasm: Option[model.JobSpecWasm],
  inputs: Option[List[model.StorageSpec]],
  outputs: Option[List[model.StorageSpec]]
)

