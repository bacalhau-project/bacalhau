package scenario

import (
	"context"
	"fmt"
	"io"
	"math/rand"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"

	"github.com/vincent-petithory/dataurl"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/storage/util"
)

// CreateSourcePath creates a file/dir path in the provided directory with a random name.
func CreateSourcePath(rootSourceDir string) (string, error) {
	// Ensure the directory exists
	if _, err := os.Stat(rootSourceDir); os.IsNotExist(err) {
		return "", fmt.Errorf("rootSourceDir does not exist: %v", err)
	}

	// Generate a random pathname
	pathname := fmt.Sprintf("input_%d", rand.Intn(1000000)) //nolint:gomnd,gosec
	return filepath.Join(rootSourceDir, pathname), nil
}

// A SetupStorage is a function that return a model.StorageSpec representing
// some data that has been prepared for use by a job. It is the responsibility
// of the function to ensure that the data has been set up correctly.
type SetupStorage func(
	ctx context.Context,
) ([]model.StorageSpec, error)

// StoredText will store the passed string as a file on the local filesystem and
// return the path to the file in a model.StorageSpec.
func StoredText(
	rootSourceDir string,
	fileContents string,
	mountPath string,
) SetupStorage {
	return func(ctx context.Context) ([]model.StorageSpec, error) {
		sourcePath, err := CreateSourcePath(rootSourceDir)
		if err != nil {
			return nil, err
		}

		// Open/create a file at the given path.
		file, err := os.Create(sourcePath)
		if err != nil {
			return nil, err
		}
		defer file.Close()

		// Write the contents to the file.
		_, err = file.WriteString(fileContents)
		if err != nil {
			return nil, err
		}

		return []model.StorageSpec{
			{
				StorageSource: model.StorageSourceLocalDirectory,
				SourcePath:    sourcePath,
				Path:          mountPath,
				Name:          mountPath,
			},
		}, nil
	}
}

// StoredFile will copy the passed file or directory into the provided mount path on the local filesystem
// and return the path to the file or directory in a model.StorageSpec.
func StoredFile(
	rootSourceDir string,
	filePath string,
	mountPath string,
) SetupStorage {
	return func(ctx context.Context) ([]model.StorageSpec, error) {
		fileInfo, err := os.Stat(filePath)
		if err != nil {
			return nil, fmt.Errorf("failed to stat file %s: %w", filePath, err)
		}

		sourcePath, err := CreateSourcePath(rootSourceDir)
		if err != nil {
			return nil, err
		}

		if fileInfo.IsDir() {
			err = copyDir(filePath, sourcePath)
			if err != nil {
				return nil, fmt.Errorf("failed to copy directory %s: %w", filePath, err)
			}
		} else {
			err = copyFile(filePath, sourcePath)
			if err != nil {
				return nil, fmt.Errorf("failed to copy file %s: %w", filePath, err)
			}
		}
		return []model.StorageSpec{
			{
				StorageSource: model.StorageSourceLocalDirectory,
				SourcePath:    sourcePath,
				Path:          mountPath,
				Name:          mountPath,
			},
		}, nil
	}
}

// InlineData will store the file directly inline in the storage spec. Unlike
// the other storage set-ups, this function loads the file immediately. This
// makes it possible to store things deeper into the Spec object without the
// test system needing to know how to prepare them.
func InlineData(data []byte) model.StorageSpec {
	return model.StorageSpec{
		StorageSource: model.StorageSourceInline,
		URL:           dataurl.EncodeBytes(data),
	}
}

// URLDownload will return a model.StorageSpec referencing a file on the passed
// HTTP test server.
func URLDownload(
	server *httptest.Server,
	urlPath string,
	mountPath string,
) SetupStorage {
	return func(_ context.Context) ([]model.StorageSpec, error) {
		finalURL, err := url.JoinPath(server.URL, urlPath)
		return []model.StorageSpec{
			{
				StorageSource: model.StorageSourceURLDownload,
				URL:           finalURL,
				Path:          mountPath,
			},
		}, err
	}
}

// ManyStores runs all of the passed setups and returns the model.StorageSpecs
// associated with all of them. If any of them fail, the error from the first to
// fail will be returned.
func ManyStores(stores ...SetupStorage) SetupStorage {
	return func(ctx context.Context) ([]model.StorageSpec, error) {
		var specs []model.StorageSpec
		for _, store := range stores {
			spec, err := store(ctx)
			if err != nil {
				return specs, err
			}
			specs = append(specs, spec...)
		}
		return specs, nil
	}
}

// copyDir copies the contents of the src directory to the dest directory.
func copyDir(src string, dest string) error {
	err := os.MkdirAll(dest, os.ModePerm)
	if err != nil {
		return fmt.Errorf("failed to create directory %s: %w", dest, err)
	}

	err = filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}

		destPath := filepath.Join(dest, relPath)

		if info.IsDir() {
			return os.MkdirAll(destPath, info.Mode())
		}

		return copyFile(path, destPath)
	})
	if err != nil {
		return err
	}

	return nil
}

// copyFile copies a file from src to dest.
func copyFile(src string, dest string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("failed to open source file %s: %w", src, err)
	}
	defer srcFile.Close()

	destFile, err := os.Create(dest)
	if err != nil {
		return fmt.Errorf("failed to create destination file %s: %w", dest, err)
	}
	defer destFile.Close()

	_, err = io.Copy(destFile, srcFile)
	if err != nil {
		return fmt.Errorf("failed to copy file from %s to %s: %w", src, dest, err)
	}

	return os.Chmod(dest, util.OS_USER_RWX)
}
