package io.swagger.server.model


/**
 * @param Entrypoint optionally override the default entrypoint
 * @param EnvironmentVariables a map of env to run the container with
 * @param Image this should be pullable by docker
 * @param WorkingDirectory working directory inside the container
 */
case class Model.JobSpecDocker (
  Entrypoint: Option[List[String]],
  EnvironmentVariables: Option[List[String]],
  Image: Option[String],
  WorkingDirectory: Option[String]
)

