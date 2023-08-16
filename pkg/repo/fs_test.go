//go:build unit || !integration

package repo

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bacalhau-project/bacalhau/pkg/config/configenv"
)

func TestNewFS(t *testing.T) {
	repo, err := NewFS(t.TempDir() + t.Name())
	require.NoError(t, err)
	require.NotNil(t, repo)

	// repo must not exists until init
	exists, err := repo.Exists()
	require.NoError(t, err)
	require.False(t, exists)

	// cannot open uninitialized repo
	err = repo.Open()
	require.Error(t, err)

	// can init a repo
	err = repo.Init(&configenv.Testing)
	require.NoError(t, err)

	// it better exist now
	exists, err = repo.Exists()
	require.NoError(t, err)
	require.True(t, exists)

	// should be able to open
	err = repo.Open()
	require.NoError(t, err)

	// cannot init an already init'ed repo.
	err = repo.Init(&configenv.Local)
	require.Error(t, err)
}
