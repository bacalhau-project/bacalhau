package io.swagger.server.model


/**
 * @param CID The unique ID of the data, where it makes sense (for example, in an IPFS storage spec this will be the data's CID). NOTE: The below is capitalized to match IPFS & IPLD (even though it's out of golang fmt) for example: ''QmTVmC7JBD2ES2qGPqBNVWnX1KeEPNrPGb7rJ8cpFgtefe''
 * @param Metadata Additional properties specific to each driver
 * @param Name Name of the spec's data, for reference. for example: ''job-9304c616-291f-41ad-b862-54e133c0149e-shard-0-host-QmdZQ7ZbhnvWY1J12XYKGHApJ6aufKyLNSvf8jZBrBaAVL''
 * @param StorageSource StorageSource is the abstract source of the data. E.g. a storage source might be a URL download, but doesn't specify how the execution engine does the download or what it will do with the downloaded data.
 * @param URL Source URL of the data
 * @param path The path that the spec's data should be mounted on, where it makes sense (for example, in a Docker storage spec this will be a filesystem path). TODO: #668 Replace with \"Path\" (note the caps) for yaml/json when we update the n.js file
 */
case class Model.StorageSpec (
  CID: Option[String],
  Metadata: Option[Map[String, String]],
  Name: Option[String],
  StorageSource: Option[Int],
  URL: Option[String],
  path: Option[String]
)

