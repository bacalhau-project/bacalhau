package validate

import (
	"os"
	"path/filepath"
)

// FileExists checks if the file at the given path exists.
// It returns an error if the file does not exist, using the provided message and arguments.
func FileExists(path string, msg string, args ...any) error {
	_, err := os.Stat(path)
	if os.IsNotExist(err) {
		return createError(msg, args...)
	}
	return nil
}

// IsFile checks if the path points to a regular file.
// It returns an error if the path is not a regular file, using the provided message and arguments.
func IsFile(path string, msg string, args ...any) error {
	info, err := os.Stat(path)
	if err != nil || !info.Mode().IsRegular() {
		return createError(msg, args...)
	}
	return nil
}

// IsDirectory checks if the path points to a directory.
// It returns an error if the path is not a directory, using the provided message and arguments.
func IsDirectory(path string, msg string, args ...any) error {
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return createError(msg, args...)
	}
	return nil
}

// HasExtension checks if the file has the specified extension.
// It returns an error if the file does not exist or does not have the given extension.
func HasExtension(path string, ext string, msg string, args ...any) error {
	if filepath.Ext(path) != ext {
		return createError(msg, args...)
	}
	return nil
}

// IsReadable checks if the file is readable by the current user.
// It returns an error if the file is not readable, using the provided message and arguments.
func IsReadable(path string, msg string, args ...any) error {
	file, err := os.Open(path)
	if err != nil {
		return createError(msg, args...)
	}
	file.Close()
	return nil
}

// IsWritable checks if the file/dir is writable by the current user.
// It returns an error if the file does not exist or is not writable.
func IsWritable(path string, msg string, args ...any) error {
	// First, check if the file exists
	info, err := os.Stat(path)
	if os.IsNotExist(err) {
		return createError(msg, args...)
	}

	// If it's a directory, check if we can create a file inside
	if info.IsDir() {
		testFile := filepath.Join(path, ".test_write_permission")
		fd, err := os.Create(testFile)
		if err != nil {
			return createError(msg, args...)
		}
		fd.Close()
		os.Remove(testFile)
		return nil
	}

	// If it's a file, try to open it in append mode
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND, 0666) //nolint:gomnd
	if err != nil {
		if os.IsPermission(err) {
			return createError(msg, args...)
		}
	} else {
		file.Close()
	}
	return nil
}

// MaxFileSize checks if the file size is not larger than the specified maximum size in bytes.
// It returns an error if the file size exceeds the maximum, using the provided message and arguments.
func MaxFileSize(path string, maxSize int64, msg string, args ...any) error {
	info, err := os.Stat(path)
	if err != nil || info.Size() > maxSize {
		return createError(msg, args...)
	}
	return nil
}
