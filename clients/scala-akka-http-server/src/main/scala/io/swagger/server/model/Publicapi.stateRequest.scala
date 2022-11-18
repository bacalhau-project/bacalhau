package io.swagger.server.model


/**
 * @param client_id  for example: ''ac13188e93c97a9c2e7cf8e86c7313156a73436036f30da1ececc2ce79f9ea51''
 * @param job_id  for example: ''9304c616-291f-41ad-b862-54e133c0149e''
 */
case class Publicapi.stateRequest (
  client_id: Option[String],
  job_id: Option[String]
)

