//go:build unit || !integration

package validate

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestFileExists(t *testing.T) {
	// Create a temporary file
	tmpFile, err := os.CreateTemp("", "example")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"Existing file", tmpFile.Name(), true},
		{"Non-existing file", "/path/to/nonexistent/file", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := FileExists(tt.path, "File does not exist: %s", tt.path)
			if tt.expected && err != nil {
				t.Errorf("FileExists() error = %v, expected no error", err)
			}
			if !tt.expected && err == nil {
				t.Errorf("FileExists() expected error for non-existent file, got nil")
			}
		})
	}
}

func TestIsFile(t *testing.T) {
	// Create a temporary file and directory
	tmpFile, err := os.CreateTemp("", "example")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	tmpDir := t.TempDir()

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"Regular file", tmpFile.Name(), true},
		{"Directory", tmpDir, false},
		{"Non-existent path", "/path/to/nonexistent/file", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := IsFile(tt.path, "Path is not a regular file: %s", tt.path)
			if tt.expected && err != nil {
				t.Errorf("IsFile() error = %v, expected no error", err)
			}
			if !tt.expected && err == nil {
				t.Errorf("IsFile() expected error, got nil")
			}
		})
	}
}

func TestIsDirectory(t *testing.T) {
	// Create a temporary file and directory
	tmpFile, err := os.CreateTemp("", "example")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	tmpDir := t.TempDir()

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"Directory", tmpDir, true},
		{"Regular file", tmpFile.Name(), false},
		{"Non-existent path", "/path/to/nonexistent/dir", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := IsDirectory(tt.path, "Path is not a directory: %s", tt.path)
			if tt.expected && err != nil {
				t.Errorf("IsDirectory() error = %v, expected no error", err)
			}
			if !tt.expected && err == nil {
				t.Errorf("IsDirectory() expected error, got nil")
			}
		})
	}
}

func TestHasExtension(t *testing.T) {
	// Create temporary files with different extensions
	tmpFile1, err := os.CreateTemp("", "example*.txt")
	require.NoError(t, err)
	defer os.Remove(tmpFile1.Name())

	tmpFile2, err := os.CreateTemp("", "example*.go")
	require.NoError(t, err)
	defer os.Remove(tmpFile2.Name())

	tests := []struct {
		name     string
		path     string
		ext      string
		expected bool
	}{
		{"Correct extension", tmpFile1.Name(), ".txt", true},
		{"Incorrect extension", tmpFile1.Name(), ".go", false},
		{"Correct extension (different file)", tmpFile2.Name(), ".go", true},
		{"No extension", "/path/to/file", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := HasExtension(tt.path, tt.ext, "File does not have extension %s: %s", tt.ext, tt.path)
			if tt.expected && err != nil {
				t.Errorf("HasExtension() error = %v, expected no error", err)
			}
			if !tt.expected && err == nil {
				t.Errorf("HasExtension() expected error, got nil")
			}
		})
	}
}

func TestIsReadable(t *testing.T) {
	// Create a temporary readable file
	tmpFile, err := os.CreateTemp("", "example")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"Readable file", tmpFile.Name(), true},
		{"Non-existent file", "/path/to/nonexistent/file", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := IsReadable(tt.path, "File is not readable: %s", tt.path)
			if tt.expected && err != nil {
				t.Errorf("IsReadable() error = %v, expected no error", err)
			}
			if !tt.expected && err == nil {
				t.Errorf("IsReadable() expected error, got nil")
			}
		})
	}
}

func TestIsWritable(t *testing.T) {
	// Create a temporary writable file
	tmpFile, err := os.CreateTemp("", "example")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	// Create a temporary directory
	tmpDir := t.TempDir()

	tests := []struct {
		name     string
		path     string
		expected bool
	}{
		{"Writable file", tmpFile.Name(), true},
		{"Writable directory", tmpDir, true},
		{"Non-existent file", "/path/to/nonexistent/file", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := IsWritable(tt.path, "File is not writable: %s", tt.path)
			if tt.expected && err != nil {
				t.Errorf("IsWritable() error = %v, expected no error", err)
			}
			if !tt.expected && err == nil {
				t.Errorf("IsWritable() expected error, got nil")
			}
		})
	}
}

func TestMaxFileSize(t *testing.T) {
	// Create a temporary file with some content
	tmpFile, err := os.CreateTemp("", "example")
	require.NoError(t, err)
	defer os.Remove(tmpFile.Name())

	content := []byte("Hello, World!")
	if _, err := tmpFile.Write(content); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	tests := []struct {
		name     string
		path     string
		maxSize  int64
		expected bool
	}{
		{"File within size limit", tmpFile.Name(), 20, true},
		{"File exceeding size limit", tmpFile.Name(), 10, false},
		{"Non-existent file", "/path/to/nonexistent/file", 100, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := MaxFileSize(tt.path, tt.maxSize, "File size exceeds maximum allowed size of %d bytes: %s", tt.maxSize, tt.path)
			if tt.expected && err != nil {
				t.Errorf("MaxFileSize() error = %v, expected no error", err)
			}
			if !tt.expected && err == nil {
				t.Errorf("MaxFileSize() expected error, got nil")
			}
		})
	}
}
