package filefs

import (
	"io/fs"
	"os"
)

type fileFs string

func New(path string) fs.FS {
	return fileFs(path)
}

// Open implements fs.FS
func (f fileFs) Open(name string) (fs.File, error) {
	if name == "." {
		return os.OpenFile(string(f), os.O_RDONLY, os.ModePerm) //nolint:gosec // G302: Read-only mode, permissions not used
	} else {
		return nil, os.ErrNotExist
	}
}
