package repo

import (
	"os"
	"path/filepath"

	"github.com/bacalhau-project/bacalhau/pkg/storage/util"
)

type ComputeStorage interface {
	// returns the root path of this compute storage
	Root() string
	// returns the namespace of the storage
	Namespace() string
	// creates a file in the namespace
	Create(name string) (*os.File, error)
	// creates a directory in the namespace
	MkdirAll(path string) error
	// removes everything managed by this storage
	RemoveAll() error
}

type fsCompStrg struct {
	namespace string
	path      string
}

func (fs *fsCompStrg) Root() string {
	return fs.path
}

func (fs *fsCompStrg) Namespace() string {
	return fs.namespace
}

func (fs *fsCompStrg) Create(name string) (*os.File, error) {
	filePath := filepath.Join(fs.path, name)
	return os.Create(filePath)
}

func (fs *fsCompStrg) MkdirAll(path string) error {
	dirPath := filepath.Join(fs.path, path)
	return os.MkdirAll(dirPath, util.OS_USER_RWX)
}

func (fs *fsCompStrg) RemoveAll() error {
	return os.RemoveAll(fs.path)
}
