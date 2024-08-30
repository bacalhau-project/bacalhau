//go:build unit || !integration

package repo

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
)

func TestNewFS(t *testing.T) {
	c, err := config.New()
	repoPath := t.TempDir() + t.Name()
	var bacCfg types.Bacalhau
	require.NoError(t, c.Unmarshal(&bacCfg))
	bacCfg.DataDir = repoPath
	require.NoError(t, err)
	repo, err := NewFS(FsRepoParams{Path: repoPath})
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
	// TODO(forrest) [refactor]: assert the repo initializes the expected values
	// in the config, such as paths and keys.
	err = repo.Init(bacCfg)
	require.NoError(t, err)

	// it better exist now
	exists, err = repo.Exists()
	require.NoError(t, err)
	require.True(t, exists)

	// should be able to open
	err = repo.Open()
	require.NoError(t, err)

	// cannot init an already init'ed repo.
	err = repo.Init(bacCfg)
	require.Error(t, err)
}
