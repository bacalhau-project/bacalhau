package io.swagger.server.model

import java.math.BigDecimal

/**
 * @param CPU cpu units for example: ''9.600000000000001''
 * @param Disk bytes for example: ''212663867801''
 * @param GPU  for example: ''1''
 * @param Memory bytes for example: ''27487790694''
 */
case class Model.ResourceUsageData (
  CPU: Option[BigDecimal],
  Disk: Option[Int],
  GPU: Option[Int],
  Memory: Option[Int]
)

