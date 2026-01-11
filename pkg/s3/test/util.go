package test

import (
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/require"
)

// AssertEqualFiles checks if two files are identical by comparing their SHA-256 hashes
func AssertEqualFiles(t *testing.T, file1, file2 string) {
	content1 := ReadFile(t, file1)
	content2 := ReadFile(t, file2)
	require.Equal(t, content1, content2, "files %s and %s are not equal", file1, file2)
}

// AssertEqualDirectories recursively compares two directories
func AssertEqualDirectories(t *testing.T, dir1, dir2 string) {
	names := func(entries []os.DirEntry) []string {
		return lo.Map(entries, func(entry fs.DirEntry, _ int) string {
			return entry.Name()
		})
	}

	entries1, err := os.ReadDir(dir1)
	require.NoError(t, err)

	entries2, err := os.ReadDir(dir2)
	require.NoError(t, err)
	require.ElementsMatch(t, names(entries1), names(entries2))

	entriesMap := make(map[string]os.DirEntry)
	for _, entry := range entries2 {
		entriesMap[entry.Name()] = entry
	}

	for _, entry1 := range entries1 {
		entry2, ok := entriesMap[entry1.Name()]
		require.True(t, ok, "file %s is not present in both directories", entry1.Name())

		path1 := filepath.Join(dir1, entry1.Name())
		path2 := filepath.Join(dir2, entry1.Name())

		if entry1.IsDir() {
			require.True(t, entry2.IsDir(), "%s is a directory but %s is a file", path1, path2)
			AssertEqualDirectories(t, path1, path2)
		} else {
			require.False(t, entry2.IsDir(), "%s is a file but %s is a directory", path1, path2)
			AssertEqualFiles(t, path1, path2)
		}
	}
}

func ReadFile(t *testing.T, path string) string {
	content, err := os.ReadFile(path) //nolint:gosec // G304: path from test fixture, controlled
	require.NoError(t, err)
	return string(content)
}
