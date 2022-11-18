package io.swagger.server.model


/**
 * @param client_id  for example: ''ac13188e93c97a9c2e7cf8e86c7313156a73436036f30da1ececc2ce79f9ea51''
 * @param id  for example: ''9304c616-291f-41ad-b862-54e133c0149e''
 * @param max_jobs  for example: ''10''
 * @param return_all 
 * @param sort_by  for example: ''created_at''
 * @param sort_reverse 
 */
case class Publicapi.listRequest (
  client_id: Option[String],
  id: Option[String],
  max_jobs: Option[Int],
  return_all: Option[Boolean],
  sort_by: Option[String],
  sort_reverse: Option[Boolean]
)

