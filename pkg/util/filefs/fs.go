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
		return os.OpenFile(string(f), os.O_RDONLY, os.ModePerm)
	} else {
		return nil, os.ErrNotExist
	}
}

func IsWritable(path string) (bool, error) {
	tmpFile := "tmpfile"

	file, err := os.CreateTemp(path, tmpFile)
	if err != nil {
		return false, err
	}

	defer os.Remove(file.Name())
	defer file.Close()

	return true, nil
}
