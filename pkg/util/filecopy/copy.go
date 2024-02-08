package filecopy

import (
	"io"
	"io/fs"
	"os"
	"path"

	"github.com/pkg/errors"
)

// File copies a single file from src to dst, preserving file mode.
// If the destination file exists, it will be overwritten.
// This may be replaced in future with
// https://github.com/golang/go/issues/62484
func CopyFile(src, dst string) error {
	var err error

	var sourceFile *os.File
	if sourceFile, err = os.Open(src); err != nil {
		return errors.Wrap(err, "failed to open source file")
	}
	defer sourceFile.Close()

	var destinationFile *os.File
	if destinationFile, err = os.Create(dst); err != nil {
		return errors.Wrap(err, "failed to open target file")
	}
	defer destinationFile.Close()

	// Efficient copying of bytes from one stream to another
	if _, err = io.Copy(destinationFile, sourceFile); err != nil {
		return errors.Wrap(err, "failed to copy file to target")
	}

	srcinfo, err := os.Stat(src)
	if err != nil {
		return errors.Wrap(err, "failed to get file mode")
	}

	err = os.Chmod(dst, srcinfo.Mode())
	if err != nil {
		return errors.Wrap(err, "failed to set file mode")
	}

	return nil
}

// CopyDir copies a whole directory from source to destination
// This may be replaced in future with
// https://github.com/golang/go/issues/62484
func CopyDir(source string, destination string) error {
	sourceInfo, err := os.Stat(source)
	if err != nil {
		return err
	}

	if err := os.MkdirAll(destination, sourceInfo.Mode()); err != nil {
		return err
	}

	var entries []fs.DirEntry
	if entries, err = os.ReadDir(source); err != nil {
		return err
	}

	for _, entry := range entries {
		src := path.Join(source, entry.Name())
		dst := path.Join(destination, entry.Name())

		if entry.IsDir() {
			if err := CopyDir(src, dst); err != nil {
				return errors.Wrap(err, "failed to copy directory")
			}
		} else {
			if err := CopyFile(src, dst); err != nil {
				return errors.Wrap(err, "failed to copy file")
			}
		}
	}
	return nil
}
