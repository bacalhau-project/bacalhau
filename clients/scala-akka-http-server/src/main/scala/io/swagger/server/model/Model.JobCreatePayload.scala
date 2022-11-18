package io.swagger.server.model


/**
 * @param ClientID the id of the client that is submitting the job
 * @param Context Optional base64-encoded tar file that will be pinned to IPFS and mounted as storage for the job. Not part of the spec so we don't flood the transport layer with it (potentially very large).
 * @param Job 
 */
case class Model.JobCreatePayload (
  ClientID: String,
  Context: Option[String],
  Job: model.Job
)

