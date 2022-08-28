package storage

// StorageVolumeConnector is how an upstream storage source will present
// the volume to a job - examples are "bind" or "library"
//
//go:generate stringer -type=StorageVolumeConnectorType --trimprefix=StorageVolumeConnector
type StorageVolumeConnectorType int

const (
	storageVolumeConnectorUnknown StorageVolumeConnectorType = iota // must be first
	StorageVolumeConnectorBind
	storageVolumeConnectorDone // must be last
)

// Used to distinguish files from directories
//
//go:generate stringer -type=FileSystemNodeType --trimprefix=FileSystemNode
type FileSystemNodeType int

const (
	fileSystemNodeUnknown FileSystemNodeType = iota // must be first
	FileSystemNodeDirectory
	FileSystemNodeFile
	fileSystemNodeDone // must be last
)
