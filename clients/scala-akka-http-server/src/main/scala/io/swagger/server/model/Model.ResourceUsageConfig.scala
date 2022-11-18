package io.swagger.server.model


/**
 * @param CPU https://github.com/BTBurke/k8sresource string
 * @param Disk 
 * @param GPU unsigned integer string
 * @param Memory github.com/c2h5oh/datasize string
 */
case class Model.ResourceUsageConfig (
  CPU: Option[String],
  Disk: Option[String],
  GPU: Option[String],
  Memory: Option[String]
)

