package io.swagger.server.model


/**
 * @param EntryPoint The name of the function in the EntryModule to call to run the job. For WASI jobs, this will always be `_start`, but jobs can choose to call other WASM functions instead. The EntryPoint must be a zero-parameter zero-result function.
 * @param EnvironmentVariables The variables available in the environment of the running program.
 * @param ImportModules TODO #880: Other WASM modules whose exports will be available as imports to the EntryModule.
 * @param Parameters The arguments supplied to the program (i.e. as ARGV).
 */
case class Model.JobSpecWasm (
  EntryPoint: Option[String],
  EnvironmentVariables: Option[Map[String, String]],
  ImportModules: Option[List[model.StorageSpec]],
  Parameters: Option[List[String]]
)

