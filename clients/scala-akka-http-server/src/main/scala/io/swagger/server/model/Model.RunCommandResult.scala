package io.swagger.server.model


/**
 * @param exitCode exit code of the run.
 * @param runnerError Runner error
 * @param stderr stderr of the run.
 * @param stderrtruncated bool describing if stderr was truncated
 * @param stdout stdout of the run. Yaml provided for `describe` output
 * @param stdouttruncated bool describing if stdout was truncated
 */
case class Model.RunCommandResult (
  exitCode: Option[Int],
  runnerError: Option[String],
  stderr: Option[String],
  stderrtruncated: Option[Boolean],
  stdout: Option[String],
  stdouttruncated: Option[Boolean]
)

