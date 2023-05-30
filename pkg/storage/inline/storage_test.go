//go:build unit || !integration

package inline

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bacalhau-project/bacalhau/pkg/model/spec/storage/inline"
	"github.com/bacalhau-project/bacalhau/pkg/storage/util"
)

func TestPlaintextInlineStorage(t *testing.T) {
	storage := NewStorage()

	tempfile := filepath.Join(t.TempDir(), "file")
	err := os.WriteFile(tempfile, []byte("test"), util.OS_ALL_RWX)
	require.NoError(t, err)

	spec, err := storage.Upload(context.Background(), tempfile)
	require.NoError(t, err)
	require.Equal(t, spec.Schema, inline.StorageType)

	size, err := storage.GetVolumeSize(context.Background(), spec)
	require.NoError(t, err)
	require.Equal(t, uint64(len("test")), size)

	root, err := storage.PrepareStorage(context.Background(), spec)
	require.NoError(t, err)

	data, err := os.ReadFile(root.Source)
	require.NoError(t, err)
	require.Equal(t, []byte("test"), data)
}

func TestDirectoryInlineStorage(t *testing.T) {
	storage := NewStorage()

	tempdir := t.TempDir()
	err := os.WriteFile(filepath.Join(tempdir, "file1"), []byte("test"), util.OS_ALL_RWX)
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(tempdir, "file2"), []byte("more"), util.OS_ALL_RWX)
	require.NoError(t, err)

	spec, err := storage.Upload(context.Background(), tempdir)
	require.NoError(t, err)
	require.Equal(t, spec.Schema, inline.StorageType)

	size, err := storage.GetVolumeSize(context.Background(), spec)
	require.NoError(t, err)
	require.Equal(t, uint64(len("test")+len("more")), size)

	root, err := storage.PrepareStorage(context.Background(), spec)
	require.NoError(t, err)

	data, err := os.ReadFile(filepath.Join(root.Source, filepath.Base(tempdir), "file1"))
	require.NoError(t, err)
	require.Equal(t, []byte("test"), data)

	data, err = os.ReadFile(filepath.Join(root.Source, filepath.Base(tempdir), "file2"))
	require.NoError(t, err)
	require.Equal(t, []byte("more"), data)
}
