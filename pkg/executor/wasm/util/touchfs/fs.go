// touchfs implements an FS.fs that will create any files that don't exist when
// they are accessed.
//
// This is necessary because our WASM implementation takes an fs.FS as an
// interface to a filesystem but the filesystem-backed FS returned by os.DirFS
// is read-only by default and will return an os.ErrNotExist if a file is opened
// for writing for the first time.
//
// To do this, touchfs tracks a directory prefix (as a string) which is both
// where it will serve and create files. Deeply nested directories are not
// supported, i.e. calling Open('a/b/c') where 'b' does not exist will still
// throw an error.
//
// This is really a limitation of the fs.FS interface and wazero shouldn't be
// using it to provide a writable filesystem.

package touchfs

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

type touchFS string

// New accepts a path to a directory and return a new fs.FS that will create
// files when they are first accessed.
func New(path string) fs.FS {
	return touchFS(path)
}

// When Open returns an error, it should be of type *PathError
// with the Op field set to "open", the Path field set to name,
// and the Err field describing the problem.
//
// Open should reject attempts to open names that do not satisfy
// ValidPath(name), returning a *PathError with Err set to
// ErrInvalid or ErrNotExist.
func (t touchFS) Open(name string) (fs.File, error) {
	dir := os.DirFS(string(t))
	file, err := dir.Open(name)

	if err != nil && errors.Is(err, os.ErrNotExist) {
		fullPath := filepath.Join(string(t), name)
		return os.Create(fullPath) //nolint:gosec // G304: fullPath constructed from module path, application controlled
	} else {
		return file, err
	}
}
