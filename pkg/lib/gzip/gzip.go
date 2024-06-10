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

// Compress compresses the sourcePath (which can be a file or a directory)
// into a gzip archive written to targetFile. It uses relative paths for
// the file headers within the archive to preserve the directory structure.
func Compress(sourcePath string, targetFile *os.File) error {
	gw := gzip.NewWriter(targetFile)
	defer gw.Close()

	tarWriter := tar.NewWriter(gw)
	defer tarWriter.Close()

	info, err := os.Stat(sourcePath)
	if err != nil {
		return fmt.Errorf("failed to stat source path: %w", err)
	}

	if info.IsDir() {
		return filepath.Walk(sourcePath, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			// Get the relative path for the file
			relpath, err := filepath.Rel(sourcePath, path)
			if err != nil {
				return err
			}
			header, err := tar.FileInfoHeader(info, "")
			if err != nil {
				return err
			}
			header.Name = relpath
			if err := tarWriter.WriteHeader(header); err != nil {
				return err
			}
			if !info.Mode().IsRegular() {
				return nil
			}
			// Open the file for reading.
			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()

			// Write the file contents to the GZIP archive.
			_, err = io.Copy(tarWriter, file)
			if err != nil {
				return err
			}

			return nil
		})
	} else {
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = filepath.Base(sourcePath)
		if err := tarWriter.WriteHeader(header); err != nil {
			return err
		}

		// Open the file for reading.
		file, err := os.Open(sourcePath)
		if err != nil {
			return err
		}
		defer file.Close()

		// Write the file contents to the GZIP archive.
		_, err = io.Copy(tarWriter, file)
		if err != nil {
			return err
		}
	}

	return nil
}

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
		if filepath.IsAbs(cleanName) {
			// Remove the common prefix (destDir) from the absolute path
			cleanName, err = filepath.Rel(filepath.Clean(destDir), cleanName)
			if err != nil {
				return fmt.Errorf("failed to compute relative path: %w", err)
			}
		}

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
