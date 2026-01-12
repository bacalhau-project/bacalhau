package mountfs

import (
	"fmt"
	"io/fs"
	"time"
)

// MountDir represents the root of a mountfs filesystem. As such, it is always a
// directory with name ".".
//
// It implements fs.FS, fs.File, fs.ReadDirFile and fs.FileInfo.
type MountDir struct {
	mounts  map[string]fs.FS
	modtime time.Time
}

// type File
func (m *MountDir) Stat() (fs.FileInfo, error) {
	return m, nil
}

func (m *MountDir) Read([]byte) (int, error) {
	return 0, &fs.PathError{
		Op:   "read",
		Path: ".",
		Err:  fmt.Errorf("is a directory"),
	}
}

func (m *MountDir) Close() error {
	return nil
}

var _ fs.File = &MountDir{}

// type ReadDirFile
func (m *MountDir) ReadDir(n int) ([]fs.DirEntry, error) {
	entries := []fs.DirEntry{}
	for name, fs := range m.mounts {
		file, err := fs.Open(".")
		if err != nil {
			return nil, err
		}
		defer func() { _ = file.Close() }()

		stat, err := file.Stat()
		if err != nil {
			return nil, err
		}

		entries = append(entries, &mountDirEntry{
			name: name,
			mode: stat.Mode(),
			dir:  m,
		})
	}

	return entries, nil
}

var _ fs.ReadDirFile = &MountDir{}

// type FileInfo
// base name of the file
func (*MountDir) Name() string {
	return "."
}

// length in bytes for regular files; system-dependent for others
func (*MountDir) Size() int64 {
	return 0
}

// file mode bits
func (*MountDir) Mode() fs.FileMode {
	return fs.ModeDir
}

// modification time
func (m *MountDir) ModTime() time.Time {
	return m.modtime
}

// abbreviation for Mode().IsDir()
func (m *MountDir) IsDir() bool {
	return m.Mode().IsDir()
}

// underlying data source (can return nil)
func (m *MountDir) Sys() any {
	return m.mounts
}

var _ fs.FileInfo = &MountDir{}
