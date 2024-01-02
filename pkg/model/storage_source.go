package model

import (
	"fmt"
)

// StorageSourceType is somewhere we can get data from
// e.g. ipfs / S3 are storage sources
// there can be multiple drivers for the same source
// e.g. ipfs fuse vs ipfs api copy
type StorageSourceType int

const (
	storageSourceUnknown StorageSourceType = iota // must be first
	StorageSourceIPFS
	StorageSourceRepoClone
	StorageSourceRepoCloneLFS
	StorageSourceURLDownload
	StorageSourceEstuary
	StorageSourceInline
	StorageSourceLocalDirectory
	StorageSourceS3
	storageSourceDone // must be last
)

var storageSourceNames = map[StorageSourceType]string{
	StorageSourceIPFS:           "ipfs",
	StorageSourceRepoClone:      "repoClone",
	StorageSourceRepoCloneLFS:   "repoCloneLFS",
	StorageSourceURLDownload:    "urlDownload",
	StorageSourceEstuary:        "estuary",
	StorageSourceInline:         "inline",
	StorageSourceLocalDirectory: "localDirectory",
	StorageSourceS3:             "s3",
}

func ParseStorageSourceType(str string) (StorageSourceType, error) {
	for typ := storageSourceUnknown + 1; typ < storageSourceDone; typ++ {
		if equal(typ.String(), str) {
			return typ, nil
		}
	}

	return storageSourceUnknown, fmt.Errorf(
		"executor: unknown source type '%s'", str)
}

func IsValidStorageSourceType(sourceType StorageSourceType) bool {
	return sourceType > storageSourceUnknown && sourceType < storageSourceDone
}

func StorageSourceTypes() []StorageSourceType {
	var res []StorageSourceType
	for typ := storageSourceUnknown + 1; typ < storageSourceDone; typ++ {
		res = append(res, typ)
	}

	return res
}

func StorageSourceNames() []string {
	var names []string
	for _, typ := range StorageSourceTypes() {
		names = append(names, typ.String())
	}
	return names
}

func (ss StorageSourceType) String() string {
	value, ok := storageSourceNames[ss]
	if !ok {
		return Unknown
	}
	return value
}

func (ss StorageSourceType) MarshalText() ([]byte, error) {
	return []byte(ss.String()), nil
}

func (ss *StorageSourceType) UnmarshalText(text []byte) (err error) {
	name := string(text)
	*ss, err = ParseStorageSourceType(name)
	return
}
