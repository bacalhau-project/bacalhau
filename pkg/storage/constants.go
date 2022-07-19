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

// StorageSourceType is somewhere we can get data from
// e.g. ipfs / S3 are storage sources
// there can be multiple drivers for the same source
// e.g. ipfs fuse vs ipfs api copy
//go:generate stringer -type=StorageSourceType --trimprefix=StorageSource
type StorageSourceType int

const (
	storageSourceUnknown StorageSourceType = iota // must be first
	StorageSourceIPFS
	StorageSourceURLDownload
	storageSourceDone // must be last
)

func ParseStorageSourceType(str string) (StorageSourceType, error) {
	for typ := storageSourceUnknown + 1; typ < storageSourceDone; typ++ {
		if equal(typ.String(), str) {
			return typ, nil
		}
	}

	return storageSourceUnknown, fmt.Errorf(
		"executor: unknown engine type '%s'", str)
}

func equal(a, b string) bool {
	a = strings.TrimSpace(a)
	b = strings.TrimSpace(b)
	return strings.EqualFold(a, b)
}

// StorageVolumeConnector is how an upstream storage source will present
// the volume to a job - examples are "bind" or "library"
//go:generate stringer -type=StorageVolumeConnectorType --trimprefix=StorageVolumeConnector
type StorageVolumeConnectorType int

const (
	storageVolumeConnectorUnknown StorageVolumeConnectorType = iota // must be first
	StorageVolumeConnectorBind
	storageVolumeConnectorDone // must be last
)

// Used to distinguish files from directories
//go:generate stringer -type=FileSystemNodeType --trimprefix=FileSystemNode
type FileSystemNodeType int

const (
	fileSystemNodeUnknown FileSystemNodeType = iota // must be first
	FileSystemNodeDirectory
	FileSystemNodeFile
	fileSystemNodeDone // must be last
)
