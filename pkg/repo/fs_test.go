//go:build unit || !integration

package repo

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bacalhau-project/bacalhau/pkg/config"
)

func TestNewFS(t *testing.T) {
	c := config.New()
	repo, err := NewFS(FsRepoParams{Path: t.TempDir() + t.Name()})
	require.NoError(t, err)
	require.NotNil(t, repo)

	// repo must not exists until init
	exists, err := repo.Exists()
	require.NoError(t, err)
	require.False(t, exists)

	// cannot open uninitialized repo
	err = repo.Open(c)
	require.Error(t, err)

	// can init a repo
	// TODO(forrest) [refactor]: assert the repo initializes the expected values
	// in the config, such as paths and keys.
	err = repo.Init(c)
	require.NoError(t, err)

	// it better exist now
	exists, err = repo.Exists()
	require.NoError(t, err)
	require.True(t, exists)

	// should be able to open
	err = repo.Open(c)
	require.NoError(t, err)

	// cannot init an already init'ed repo.
	err = repo.Init(c)
	require.Error(t, err)
}
