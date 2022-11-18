package io.swagger.server.model


/**
 * @param BatchSize how many \"items\" are to be processed in each shard we first apply the glob pattern which will result in a flat list of items this number decides how to group that flat list into actual shards run by compute nodes
 * @param GlobPattern divide the inputs up into the smallest possible unit for example /_* would mean \"all top level files or folders\" this being an empty string means \"no sharding\"
 * @param GlobPatternBasePath when using multiple input volumes what path do we treat as the common mount path to apply the glob pattern to
 */
case class Model.JobShardingConfig (
  BatchSize: Option[Int],
  GlobPattern: Option[String],
  GlobPatternBasePath: Option[String]
)

