package gzip

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const DefaultMaxDecompressSize = 100 * 1024 * 1024 * 1024 // 100 GB

// Decompress takes the path to a .tar.gz file and decompresses it into the specified directory.
func Decompress(tarGzPath, destDir string) error {
	return DecompressWithMaxBytes(tarGzPath, destDir, DefaultMaxDecompressSize)
}

// DecompressWithMaxBytes takes the path to a .tar.gz file and decompresses it into the specified directory.
// It enforces a maximum decompressed file size to prevent decompression bombs.
func DecompressWithMaxBytes(tarGzPath, destDir string, maxDecompressSize int64) error {
	// Open the tar.gz file for reading.
	file, err := os.Open(tarGzPath)
	if err != nil {
		return err
	}
	defer file.Close()

	// Create a gzip reader.
	gzr, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzr.Close()

	// Create a tar reader.
	tr := tar.NewReader(gzr)

	// Iterate through the files in the archive.
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break // End of archive
		}
		if err != nil {
			return err
		}

		// Clean the name to mitigate directory traversal
		cleanName := filepath.Clean(header.Name)
		if strings.HasPrefix(cleanName, ".."+string(filepath.Separator)) {
			return fmt.Errorf("invalid file path: %s", header.Name)
		}

		// Construct the full path for the file to be written to.
		target := filepath.Join(destDir, cleanName)

		// Check the file type.
		switch header.Typeflag {
		case tar.TypeDir:
			// It's a directory; create it.
			if err := os.MkdirAll(target, os.FileMode(header.Mode)); err != nil {
				return err
			}
		case tar.TypeReg:
			// It's a file; create it, preserving the file mode.
			// Check for decompression bomb
			if header.Size > maxDecompressSize {
				return fmt.Errorf("file too large: %s", header.Name)
			}
			fileToWrite, err := os.OpenFile(target, os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return err
			}
			// Copy the file data from the archive, enforcing the file size limit.
			if _, err := io.CopyN(fileToWrite, tr, header.Size); err != nil {
				fileToWrite.Close()
				return err
			}
			fileToWrite.Close()
		}
	}

	return nil
}

// DecompressInPlace takes the path to a .tar.gz file and decompresses it into the same directory.
func DecompressInPlace(tarGzPath string) (string, error) {
	uncompressedPath := strings.TrimSuffix(tarGzPath, ".tar.gz")
	err := Decompress(tarGzPath, uncompressedPath)
	if err != nil {
		return "", err
	}
	return uncompressedPath, nil
}
