package targzip

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/c2h5oh/datasize"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/util/closer"
)

const (
	MaximumContextSize            datasize.ByteSize = 10 * datasize.MB
	worldReadOwnerWritePermission fs.FileMode       = 0755
)

func Compress(ctx context.Context, src string, buf io.Writer) error {
	return compress(ctx, src, buf, MaximumContextSize)
}

func Decompress(src io.Reader, dst string) error {
	return decompress(src, dst, MaximumContextSize)
}

func UncompressedSize(src io.Reader) (datasize.ByteSize, error) {
	var size datasize.ByteSize
	zr, err := gzip.NewReader(src)
	if err != nil {
		return 0, err
	}
	tr := tar.NewReader(zr)

	var header *tar.Header
	for header, err = tr.Next(); err == nil; header, err = tr.Next() {
		size += datasize.ByteSize(header.Size)
	}
	if err == io.EOF {
		err = nil
	}
	return size, err
}

// from https://github.com/mimoo/eureka/blob/master/folders.go under Apache 2
//
//nolint:gocyclo
func compress(ctx context.Context, src string, buf io.Writer, max datasize.ByteSize) error {
	_, span := system.NewSpan(ctx, system.GetTracer(), "pkg/util/targzip.compress")
	defer span.End()

	// tar > gzip > buf
	zr := gzip.NewWriter(buf)
	tw := tar.NewWriter(zr)

	// is file a folder?
	fi, err := os.Stat(src)
	if err != nil {
		return err
	}
	mode := fi.Mode()
	if mode.IsRegular() {
		if fi.Size() > int64(max) {
			return fmt.Errorf("file %s bigger than max size %s", src, max.HumanReadable())
		}
		// get header
		var header *tar.Header
		header, err = tar.FileInfoHeader(fi, src)
		if err != nil {
			return err
		}
		// write header
		if err = tw.WriteHeader(header); err != nil { //nolint:gocritic
			return err
		}
		// get content
		var data *os.File
		data, err = os.Open(src)
		if err != nil {
			return err
		}
		defer closer.CloseWithLogOnError(fi.Name(), data)
		if _, err = io.Copy(tw, data); err != nil {
			return err
		}
	} else if mode.IsDir() { // folder
		// walk through every file in the folder
		err = filepath.Walk(src, func(file string, fi os.FileInfo, _ error) error {
			// generate tar header
			var header *tar.Header
			header, err = tar.FileInfoHeader(fi, file)
			if err != nil {
				return err
			}

			// must provide real name
			// (see https://golang.org/src/archive/tar/common.go?#L626)
			header.Name = filepath.ToSlash(file)

			// write header
			if err = tw.WriteHeader(header); err != nil { //nolint:gocritic
				return err
			}
			// if not a dir, write file content
			if !fi.IsDir() {
				var data *os.File
				var fi os.FileInfo
				fi, err = os.Stat(file)
				if err != nil {
					return err
				}
				if fi.Size() > int64(max) {
					return fmt.Errorf("file %s bigger than max size %s", file, max.HumanReadable())
				}
				data, err = os.Open(file)
				if err != nil {
					return err
				}
				if _, err = io.Copy(tw, data); err != nil { //nolint:gocritic
					return err
				}
				closer.CloseWithLogOnError(fi.Name(), data)
			}
			return nil
		})
		if err != nil {
			return err
		}
	} else {
		return fmt.Errorf("error: file type not supported")
	}

	// produce tar
	if err := tw.Close(); err != nil {
		return err
	}
	// produce gzip
	if err := zr.Close(); err != nil {
		return err
	}
	//
	return nil
}

func decompress(src io.Reader, dst string, max datasize.ByteSize) error {
	// ensure destination directory exists
	err := os.Mkdir(dst, worldReadOwnerWritePermission)
	if err != nil {
		return err
	}

	// ungzip
	zr, err := gzip.NewReader(src)
	if err != nil {
		return err
	}
	// untar
	tr := tar.NewReader(zr)

	// uncompress each element
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			return err
		}
		target := header.Name

		// validate name against path traversal
		if !validRelPath(header.Name) {
			return fmt.Errorf("tar contained invalid name error %q", target)
		}

		// add dst + re-format slashes according to system
		target, err = sanitizeArchivePath(dst, header.Name)
		if err != nil {
			return err
		}
		// if no join is needed, replace with ToSlash:
		// target = filepath.ToSlash(header.Name)

		// check the type
		switch header.Typeflag {
		// if its a dir and it doesn't exist create it (with 0755 permission)
		case tar.TypeDir:
			if _, err := os.Stat(target); err != nil {
				if err := os.MkdirAll(target, worldReadOwnerWritePermission); err != nil {
					return err
				}
			}
		// if it's a file create it (with same permission)
		case tar.TypeReg:
			if header.Size > int64(max) {
				return fmt.Errorf("file %s bigger than max size %s", header.Name, max.HumanReadable())
			}
			fileToWrite, err := os.OpenFile(target, os.O_CREATE|os.O_RDWR, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			// copy over contents (max 10MB per file!)
			if _, err := io.CopyN(fileToWrite, tr, int64(max)); err != nil { //nolint:gomnd
				// io.EOF is expected
				if err != io.EOF {
					return err
				}
			}
			// manually close here after each file operation; defering would cause each file close
			// to wait until all operations have completed.
			fileToWrite.Close()
		}
	}

	//
	return nil
}

// check for path traversal and correct forward slashes
func validRelPath(p string) bool {
	if p == "" || strings.Contains(p, `\`) || strings.HasPrefix(p, "/") || strings.Contains(p, "../") {
		return false
	}
	return true
}

// Sanitize archive file pathing from "G305: Zip Slip vulnerability"
func sanitizeArchivePath(d, t string) (v string, err error) {
	v = filepath.Join(d, t)
	if strings.HasPrefix(v, filepath.Clean(d)) {
		return v, nil
	}

	return "", fmt.Errorf("%s: %s", "content filepath is tainted", t)
}
