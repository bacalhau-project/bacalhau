package storage

import (
	"fmt"
	"strings"
)

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
