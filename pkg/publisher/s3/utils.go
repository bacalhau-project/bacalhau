package s3

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/model"
)

func ParsePublishedKey(key string, executionID string, job model.Job, archive bool) string {
	if archive && !strings.HasSuffix(key, ".tar.gz") {
		key = key + ".tar.gz"
	}
	if !archive && !strings.HasSuffix(key, "/") {
		key = key + "/"
	}
	key = strings.ReplaceAll(key, "{executionID}", executionID)
	key = strings.ReplaceAll(key, "{jobID}", job.ID())
	key = strings.ReplaceAll(key, "{date}", time.Now().Format("YYYYMMDD"))
	key = strings.ReplaceAll(key, "{time}", time.Now().Format("HHMMSS"))
	return key
}

func archiveDirectory(sourceDir string, targetFile *os.File) error {
	gw := gzip.NewWriter(targetFile)
	defer gw.Close()

	tarWriter := tar.NewWriter(gw)
	defer tarWriter.Close()

	return filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		// Get the relative path for the file
		relpath, err := filepath.Rel(sourceDir, path)
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
}
