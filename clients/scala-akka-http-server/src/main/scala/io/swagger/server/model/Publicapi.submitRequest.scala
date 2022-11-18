package io.swagger.server.model


/**
 * @param client_public_key The base64-encoded public key of the client:
 * @param data 
 * @param signature A base64-encoded signature of the data, signed by the client:
 */
case class Publicapi.submitRequest (
  client_public_key: String,
  data: model.JobCreatePayload,
  signature: String
)

