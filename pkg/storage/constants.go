package storage

// StorageSourceType is somewhere we can get data from
// e.g. ipfs / S3 are storage sources
// there can be multiple drivers for the same source
// e.g. ipfs fuse vs ipfs api copy
//go:generate stringer -type=StorageSourceType --trimprefix=StorageSource
type StorageSourceType int

const (
	storageSourceUnknown StorageSourceType = iota // must be first
	StorageSourceIPFS
	storageSourceDone // must be last
)

const IPFSFuseDocker = "ipfs_fuse"
const IPFSAPICopy = "ipfs_copy"
const IPFSDefault = "ipfs"

const URLDownload = "url_download"
