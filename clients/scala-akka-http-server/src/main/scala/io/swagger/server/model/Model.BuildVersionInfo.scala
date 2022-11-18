package io.swagger.server.model


/**
 * @param builddate  for example: ''2022-11-16T14:03:31Z''
 * @param gitcommit  for example: ''d612b63108f2b5ce1ab2b9e02444eb1dac1d922d''
 * @param gitversion  for example: ''v0.3.12''
 * @param goarch  for example: ''amd64''
 * @param goos  for example: ''linux''
 * @param major  for example: ''0''
 * @param minor  for example: ''3''
 */
case class Model.BuildVersionInfo (
  builddate: Option[String],
  gitcommit: Option[String],
  gitversion: Option[String],
  goarch: Option[String],
  goos: Option[String],
  major: Option[String],
  minor: Option[String]
)

