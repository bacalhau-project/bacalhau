package compute

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/bacalhau-project/bacalhau/pkg/repo"
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

func NewRegistrationFile(prefix string, strg repo.ComputeStorage) *RegistrationFile {
	regFilename := fmt.Sprintf("%s.registration.lock", prefix)
	regFilename = filepath.Join(strg.Root(), regFilename)
	return &RegistrationFile{
		path: regFilename,
	}
}

func (r *RegistrationFile) Exists() bool {
	_, err := os.Stat(r.path)
	return !os.IsNotExist(err)
}

func (r *RegistrationFile) Set() error {
	return os.WriteFile(r.path, []byte{}, fileMode)
}
