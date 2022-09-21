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
	engine, err := EnsureStorageSourceType(spec.Engine, spec.EngineName)
	if err != nil {
		return spec, err
	}
	spec.Engine = engine
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

// StorageSpec represents some data on a storage engine. Storage engines are
// specific to particular execution engines, as different execution engines
// will mount data in different ways.
type StorageSpec struct {
	// TODO: #645 Is this engine name the same as the Job EngineName?
	// Engine is the execution engine that can mount the spec's data.
	Engine     StorageSourceType `json:"Engine,omitempty" yaml:"Engine,omitempty"`
	EngineName string            `json:"EngineName,omitempty" yaml:"EngineName,omitempty"`

	// Name of the spec's data, for reference.
	Name string `json:"Name,omitempty" yaml:"Name,omitempty"`

	// The unique ID of the data, where it makes sense (for example, in an
	// IPFS storage spec this will be the data's Cid).
	// NOTE: The below is capitalized to match IPFS & IPLD (even thoough it's out of golang fmt)
	Cid string `json:"Cid,omitempty" yaml:"Cid,omitempty"`

	// Source URL of the data
	URL string `json:"URL,omitempty" yaml:"URL,omitempty"`

	// The path that the spec's data should be mounted on, where it makes
	// sense (for example, in a Docker storage spec this will be a filesystem
	// path).
	Path string `json:"Path,omitempty" yaml:"Path,omitempty"`

	// Additional properties specific to each driver
	Metadata map[string]string `json:"Metadata,omitempty" yaml:"Metadata,omitempty"`
}
