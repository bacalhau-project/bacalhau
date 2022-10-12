// mountfs provides a fs.FS that can 'mount' other filesystems at a directory
// prefix. This is purely internal to Go and not related to the similar kernel
// concept. This allows composition of fs.FS objects in a virtual hierarchy.
// mountfs can be nested arbitrarily deeply to provide whatever level of
// hierarchy is required.
//
// To use it, call Mount on the object returned from New:
//
//      fs := mountfs.New()
//      fs.Mount("root", os.DirFS("/"))
//      // Files from "/" now appear under "/root/"
//		fs.Open("/root/cool.txt") // actually opens /root.txt

package mountfs

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// New returns an empty filesystem with no mounts.
func New() *MountDir {
	return &MountDir{
		mounts:  map[string]fs.FS{},
		modtime: time.Time{},
	}
}

func (m *MountDir) Open(name string) (fs.File, error) {
	path := filepath.Clean(name)
	if path == "." {
		return m, nil
	}

	pathComponents := strings.Split(path, string(os.PathSeparator))
	pathPrefix, restOfPath := pathComponents[0], pathComponents[1:]

	// Find the mounted file system to delegate to. If the prefix is not in the
	// map, that is effectively the same as the directory not existing.
	mountedFs, exists := m.mounts[pathPrefix]
	if !exists {
		return nil, os.ErrNotExist
	}

	// Pass the rest of the path to the delegate FS.
	subPath := filepath.Clean(strings.Join(restOfPath, string(os.PathSeparator)))
	return mountedFs.Open(subPath)
}

// Mount makes the files available in filesystem available under the "/prefix"
func (m *MountDir) Mount(prefix string, filesystem fs.FS) error {
	if strings.ContainsRune(prefix, os.PathSeparator) {
		return fmt.Errorf("cannot deep mount '%s' – chain multiple mountfs instead", prefix)
	}

	_, exists := m.mounts[prefix]
	if exists {
		return fmt.Errorf("a filesystem with prefix '%s' is already mounted", prefix)
	}

	m.mounts[prefix] = filesystem
	return nil
}

// Unmount makes a previously existing "/prefix" no longer serve files
func (m *MountDir) Unmount(prefix string) error {
	_, exists := m.mounts[prefix]
	if !exists {
		return fmt.Errorf("a filesystem with prefix '%s' is not mounted", prefix)
	}

	delete(m.mounts, prefix)
	return nil
}

var _ fs.FS = New()
