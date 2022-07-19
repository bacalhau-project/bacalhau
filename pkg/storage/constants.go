package storage

//go:generate stringer -type=StorageVolumeType --trimprefix=StorageVolume
type StorageVolumeType int

const (
	storageVolumeUnknown StorageVolumeType = iota // must be first
	StorageVolumeBind
	storageVolumeDone // must be last
)

//go:generate stringer -type=IPFSNodeType --trimprefix=IPFSNode
type IPFSNodeType int

const (
	ipfsNodeUnknown IPFSType = iota // must be first
	IPFSNodeDirectory
	IPFSNodeFile
	ipfsNodeDone // must be last
)

//go:generate stringer -type=IPFSType --trimprefix=IPFS
type IPFSType int

const (
	ipfsUnknown IPFSType = iota // must be first
	IPFSDirectory
	IPFSFile
	ipfsDone // must be last
)

const StorageVolumeTypeBind = "bind"

const IPFSTypeDirectory = "directory"
const IPFSTypeFile = "file"

const IPFSFuseDocker = "ipfs_fuse"
const IPFSAPICopy = "ipfs_copy"
const IPFSDefault = "ipfs"
