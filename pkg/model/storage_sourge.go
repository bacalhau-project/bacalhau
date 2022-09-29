package model

import (
	"fmt"
)

// StorageSourceType is somewhere we can get data from
// e.g. ipfs / S3 are storage sources
// there can be multiple drivers for the same source
// e.g. ipfs fuse vs ipfs api copy
//
//go:generate stringer -type=StorageSourceType --trimprefix=StorageSource
type StorageSourceType int

const (
	storageSourceUnknown StorageSourceType = iota // must be first
	StorageSourceIPFS
	StorageSourceURLDownload
	StorageSourceFilecoinUnsealed
	StorageSourceFilecoin
	StorageSourceEstuary
	storageSourceDone // must be last
)

func ParseStorageSourceType(str string) (StorageSourceType, error) {
	for typ := storageSourceUnknown + 1; typ < storageSourceDone; typ++ {
		if equal(typ.String(), str) {
			return typ, nil
		}
	}

	return storageSourceUnknown, fmt.Errorf(
		"executor: unknown source type '%s'", str)
}

func EnsureStorageSourceType(typ StorageSourceType, str string) (StorageSourceType, error) {
	if IsValidStorageSourceType(typ) {
		return typ, nil
	}
	return ParseStorageSourceType(str)
}

func EnsureStorageSpecSourceType(spec StorageSpec) (StorageSpec, error) {
	engine, err := EnsureStorageSourceType(spec.StorageSource, spec.StorageSourceName)
	if err != nil {
		return spec, err
	}
	spec.StorageSource = engine
	return spec, nil
}

func EnsureStorageSpecsSourceTypes(specs []StorageSpec) ([]StorageSpec, error) {
	ret := []StorageSpec{}
	for _, spec := range specs {
		newSpec, err := EnsureStorageSpecSourceType(spec)
		if err != nil {
			return ret, err
		}
		ret = append(ret, newSpec)
	}
	return ret, nil
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
func (ss StorageSourceType) MarshalText() ([]byte, error) {
	return []byte(ss.String()), nil
}

func (ss *StorageSourceType) UnmarshalText(text []byte) (err error) {
	name := string(text)
	*ss, err = ParseStorageSourceType(name)
	return
}
