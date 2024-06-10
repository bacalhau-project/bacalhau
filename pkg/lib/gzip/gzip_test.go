//go:build unit || !integration

package gzip_test

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/bacalhau-project/bacalhau/pkg/lib/gzip"
)

type GzipTestSuite struct {
	suite.Suite
}

func (suite *GzipTestSuite) createTestFiles(baseDir string) error {
	files := map[string]string{
		"file1.txt": "Hello World",
		"file2.txt": "This is a test file",
	}

	for name, content := range files {
		path := filepath.Join(baseDir, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return err
		}
	}

	return nil
}

func (suite *GzipTestSuite) createNestedTestFiles(baseDir string) error {
	dirs := []string{"subDir1", "subDir2"}
	files := map[string]string{
		"subDir1/file1.txt": "Nested file 1",
		"subDir2/file2.txt": "Nested file 2",
	}

	for _, dir := range dirs {
		if err := os.Mkdir(filepath.Join(baseDir, dir), 0755); err != nil {
			return err
		}
	}

	for name, content := range files {
		path := filepath.Join(baseDir, name)
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			return err
		}
	}

	return nil
}

func (suite *GzipTestSuite) verifyTestFiles(baseDir string) {
	files := map[string]string{
		"file1.txt": "Hello World",
		"file2.txt": "This is a test file",
	}

	for name, expectedContent := range files {
		path := filepath.Join(baseDir, name)
		content, err := os.ReadFile(path)
		suite.Require().NoError(err, "Failed to read file %s", name)
		suite.Require().Equal(expectedContent, string(content), "File content mismatch for %s", name)
	}
}

func (suite *GzipTestSuite) verifyNestedTestFiles(baseDir string) {
	files := map[string]string{
		"subDir1/file1.txt": "Nested file 1",
		"subDir2/file2.txt": "Nested file 2",
	}

	for name, expectedContent := range files {
		path := filepath.Join(baseDir, name)
		content, err := os.ReadFile(path)
		suite.Require().NoError(err, "Failed to read file %s", name)
		suite.Require().Equal(expectedContent, string(content), "File content mismatch for %s", name)
	}
}

func (suite *GzipTestSuite) verifyRelativePaths(sourceDir, outputDir string) {
	// Walk the source directory and verify each file exists in the decompressed output with the same relative path
	err := filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}
		decompressedPath := filepath.Join(outputDir, relPath)
		if info.IsDir() {
			suite.Require().DirExists(decompressedPath, "Directory %s should exist in decompressed output", decompressedPath)
		} else {
			suite.Require().FileExists(decompressedPath, "File %s should exist in decompressed output", decompressedPath)
		}
		return nil
	})
	suite.Require().NoError(err, "Failed to verify relative paths")
}

func (suite *GzipTestSuite) TestCompressDecompressFile() {
	sourceFile := filepath.Join(suite.T().TempDir(), "testFile.txt")
	outputFile := filepath.Join(suite.T().TempDir(), "testFile.tar.gz")
	outputDir := suite.T().TempDir()

	// Create test file
	err := os.WriteFile(sourceFile, []byte("This is a test file"), 0644)
	suite.Require().NoError(err, "Failed to create test file")

	// Create target output file
	targetFile, err := os.Create(outputFile)
	suite.Require().NoError(err, "Failed to create target file")

	// Compress the source file
	err = gzip.Compress(sourceFile, targetFile)
	suite.Require().NoError(err, "Failed to compress file")
	targetFile.Close()

	// Decompress the tar.gz file
	err = gzip.Decompress(outputFile, outputDir)
	suite.Require().NoError(err, "Failed to decompress file")

	// Verify decompressed file
	content, err := os.ReadFile(filepath.Join(outputDir, "testFile.txt"))
	suite.Require().NoError(err, "Failed to read decompressed file")
	suite.Require().Equal("This is a test file", string(content), "Decompressed file content mismatch")
}

func (suite *GzipTestSuite) TestCompressDecompress() {
	sourceDir := suite.T().TempDir()
	outputDir := suite.T().TempDir()
	outputFile := filepath.Join(suite.T().TempDir(), "testDir.tar.gz")

	// Create test directory and files
	err := suite.createTestFiles(sourceDir)
	suite.Require().NoError(err, "Failed to create test files")

	// Create target output file
	targetFile, err := os.Create(outputFile)
	suite.Require().NoError(err, "Failed to create target file")

	// Compress the source directory
	err = gzip.Compress(sourceDir, targetFile)
	suite.Require().NoError(err, "Failed to compress directory")
	targetFile.Close()

	// Decompress the tar.gz file
	err = gzip.Decompress(outputFile, outputDir)
	suite.Require().NoError(err, "Failed to decompress file")

	// Verify decompressed files
	suite.verifyTestFiles(outputDir)

	// Verify relative paths
	suite.verifyRelativePaths(sourceDir, outputDir)
}

func (suite *GzipTestSuite) TestCompressDecompressNested() {
	sourceDir := suite.T().TempDir()
	outputDir := suite.T().TempDir()
	outputFile := filepath.Join(suite.T().TempDir(), "nestedDir.tar.gz")

	// Create test directory and files
	err := suite.createNestedTestFiles(sourceDir)
	suite.Require().NoError(err, "Failed to create nested test files")

	// Create target output file
	targetFile, err := os.Create(outputFile)
	suite.Require().NoError(err, "Failed to create target file")

	// Compress the source directory
	err = gzip.Compress(sourceDir, targetFile)
	suite.Require().NoError(err, "Failed to compress directory")
	targetFile.Close()

	// Decompress the tar.gz file
	err = gzip.Decompress(outputFile, outputDir)
	suite.Require().NoError(err, "Failed to decompress file")

	// Verify decompressed files
	suite.verifyNestedTestFiles(outputDir)

	// Verify relative paths
	suite.verifyRelativePaths(sourceDir, outputDir)
}

func TestGzipTestSuite(t *testing.T) {
	suite.Run(t, new(GzipTestSuite))
}
