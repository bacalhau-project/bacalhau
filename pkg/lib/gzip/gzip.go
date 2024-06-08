package gzip

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"
)

const DefaultMaxDecompressSize = 100 * 1024 * 1024 * 1024 // 100 GB

// Decompress takes the path to a .tar.gz file and decompresses it into the specified directory.
func Decompress(tarGzPath, destDir string) error {
	log.Debug().Msgf("Decompressing %s to %s", tarGzPath, destDir)
	return DecompressWithMaxBytes(tarGzPath, destDir, DefaultMaxDecompressSize)
}

// DecompressWithMaxBytes takes the path to a .tar.gz file and decompresses it into the specified directory.
// It enforces a maximum decompressed file size (per file) to prevent decompression bombs.
func DecompressWithMaxBytes(tarGzPath, destDir string, maxDecompressSize int64) error {
	fmt.Printf("Decompressing %s to %s\n", tarGzPath, destDir)

	// Open the tar.gz file for reading.
	file, err := os.Open(tarGzPath)
	if err != nil {
		return fmt.Errorf("failed to open tar.gz file: %w", err)
	}
	defer file.Close()

	// Create a gzip reader.
	gzr, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
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
			return fmt.Errorf("failed to read tar header: %w", err)
		}

		// Clean the name to mitigate directory traversal
		cleanName := filepath.Clean(header.Name)
		if strings.HasPrefix(cleanName, ".."+string(filepath.Separator)) {
			return fmt.Errorf("invalid file path: %s", header.Name)
		}

		// Construct the full path for the file to be written to.
		target := filepath.Join(destDir, cleanName)
		fmt.Printf("Processing %s\n", target)

		// Check the file type.
		switch header.Typeflag {
		case tar.TypeDir:
			// It's a directory; create it.
			fmt.Printf("Creating directory: %s\n", target)
			if err := os.MkdirAll(target, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("failed to create directory: %w", err)
			}
		case tar.TypeReg:
			// It's a file; create it, preserving the file mode.
			// Check for decompression bomb
			if header.Size > maxDecompressSize {
				return fmt.Errorf("file too large: %s", header.Name)
			}
			fmt.Printf("Creating file: %s\n", target)
			fileToWrite, err := os.OpenFile(target, os.O_RDWR|os.O_CREATE|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return fmt.Errorf("failed to create file: %w", err)
			}
			// Copy the file data from the archive, enforcing the file size limit.
			if _, err := io.CopyN(fileToWrite, tr, header.Size); err != nil {
				fileToWrite.Close()
				return fmt.Errorf("failed to copy file data: %w", err)
			}
			fileToWrite.Close()
		}
	}

	fmt.Println("Decompression completed successfully")
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
