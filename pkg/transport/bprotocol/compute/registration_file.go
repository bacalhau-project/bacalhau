package compute

import (
	"os"
)

const (
	fileMode = 0644
)

// RegistrationFile is a sentinel on disk whose presence
// is used to denote that a node has successfully registered
// with the requester. This file is per-node to allow multiple
// compute nodes using the same shared directory for config.
type RegistrationFile struct {
	path string
}

func NewRegistrationFile(path string) *RegistrationFile {
	return &RegistrationFile{
		path: path,
	}
}

func (r *RegistrationFile) Exists() bool {
	_, err := os.Stat(r.path)
	return !os.IsNotExist(err)
}

func (r *RegistrationFile) Set() error {
	return os.WriteFile(r.path, []byte{}, fileMode)
}
