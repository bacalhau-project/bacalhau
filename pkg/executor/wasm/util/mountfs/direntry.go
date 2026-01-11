package mountfs

import (
	"io/fs"
	"os"
)

// mountDirEntry represents a directory that is the root of one of the mounted
// filesystems in a MountDir. It will be returned from MountDir.ReadDir to list
// the contents of the mountfs.
//
// It caches the fs.FileMode of the tip of the mounted FS because doing an Open
// on the FS to Stat the root is a potentially erroring operation, but the
// accessors on fs.DirEntry cannot return errors.
type mountDirEntry struct {
	name string
	mode fs.FileMode
	dir  *MountDir
}

// Name returns the name of the file (or subdirectory) described by the entry.
// This name is only the final element of the path (the base name), not the entire path.
// For example, Name would return "hello.go" not "home/gopher/hello.go".
func (de *mountDirEntry) Name() string {
	return de.name
}

// IsDir reports whether the entry describes a directory.
func (de *mountDirEntry) IsDir() bool {
	return de.mode.IsDir()
}

// Type returns the type bits for the entry.
// The type bits are a subset of the usual FileMode bits, those returned by the FileMode.Type method.
func (de *mountDirEntry) Type() fs.FileMode {
	return de.mode
}

// Info returns the FileInfo for the file or subdirectory described by the entry.
// The returned FileInfo may be from the time of the original directory read
// or from the time of the call to Info. If the file has been removed or renamed
// since the directory read, Info may return an error satisfying errors.Is(err, ErrNotExist).
// If the entry denotes a symbolic link, Info reports the information about the link itself,
// not the link's target.
func (de *mountDirEntry) Info() (fs.FileInfo, error) {
	fs, exists := de.dir.mounts[de.name]
	if !exists {
		return nil, os.ErrNotExist
	}

	file, err := fs.Open(".")
	if err != nil {
		return nil, err
	}
	defer func() { _ = file.Close() }()

	return file.Stat()
}

var _ fs.DirEntry = &mountDirEntry{}
