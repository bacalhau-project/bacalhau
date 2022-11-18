package io.swagger.server.model


/**
 * @param Command optional program specified on commandline, like python -c \"print(1+1)\"
 * @param DeterministicExecution must this job be run in a deterministic context?
 * @param JobContext 
 * @param Language e.g. python
 * @param LanguageVersion e.g. 3.8
 * @param ProgramPath optional program path relative to the context dir. one of Command or ProgramPath must be specified
 * @param RequirementsPath optional requirements.txt (or equivalent) path relative to the context dir
 */
case class Model.JobSpecLanguage (
  Command: Option[String],
  DeterministicExecution: Option[Boolean],
  JobContext: Option[model.StorageSpec],
  Language: Option[String],
  LanguageVersion: Option[String],
  ProgramPath: Option[String],
  RequirementsPath: Option[String]
)

