package io.swagger.server.model


/**
 * @param IPFSMount 
 * @param root 
 * @param tmp 
 */
case class Types.FreeSpace (
  IPFSMount: Option[types.MountStatus],
  root: Option[types.MountStatus],
  tmp: Option[types.MountStatus]
)

