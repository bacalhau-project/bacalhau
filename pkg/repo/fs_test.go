//go:build unit || !integration

package repo_test

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/repo"
)

func TestNewFSErrorCases(t *testing.T) {
	c, err := config.New()
	r, err := repo.NewFS(repo.FsRepoParams{
		Path: t.TempDir() + t.Name(),
	})
	require.NoError(t, err)
	require.NotNil(t, r)

	// repo must not exists until init
	exists, err := r.Exists()
	require.NoError(t, err)
	require.False(t, exists)

	repoPath, err := r.Path()
	require.Error(t, err)
	require.Empty(t, repoPath)

	// cannot open uninitialized repo
	err = r.Open(c)
	require.Error(t, err)

	// cannot read the repo child directories and files
	d, err := r.OrchestratorDir()
	require.Error(t, err)
	require.Empty(t, d)

	d, err = r.NetworkTransportDir()
	require.Error(t, err)
	require.Empty(t, d)

	d, err = r.ComputeDir()
	require.Error(t, err)
	require.Empty(t, d)

	d, err = r.ExecutionDir()
	require.Error(t, err)
	require.Empty(t, d)

	// nor the repo files
	f, err := r.UserKeyPath()
	require.Error(t, err)
	require.Empty(t, f)

	// can init a repo
	err = r.Init(c)
	require.NoError(t, err)

	// cannot init an already init'ed repo.
	err = r.Init(c)
	require.Error(t, err)

}

func TestNewFSHybridNode(t *testing.T) {
	r, err := repo.NewFS(repo.FsRepoParams{
		Path: t.TempDir() + t.Name(),
	})
	require.NoError(t, err)
	require.NotNil(t, r)

	c, err := config.New()
	require.NoError(t, err)
	// can init a repo
	err = r.Init(c)
	require.NoError(t, err)

	// it better exist now
	exists, err := r.Exists()
	require.NoError(t, err)
	require.True(t, exists)

	// can get the path
	repoPath, err := r.Path()
	require.NoError(t, err)
	require.NotEmpty(t, repoPath)

	// can read the repo child directories and files
	d, err := r.OrchestratorDir()
	require.NoError(t, err)
	require.Equal(t, filepath.Join(repoPath, repo.OrchestratorDirKey), d)

	d, err = r.NetworkTransportDir()
	require.NoError(t, err)
	require.Equal(t, filepath.Join(repoPath, repo.NetworkTransportDirKey), d)

	d, err = r.ComputeDir()
	require.NoError(t, err)
	require.Equal(t, filepath.Join(repoPath, repo.ComputeDirKey), d)

	d, err = r.ExecutionDir()
	require.NoError(t, err)
	require.Equal(t, filepath.Join(repoPath, repo.ExecutionDirKey), d)

	f, err := r.UserKeyPath()
	require.NoError(t, err)
	require.NotEmpty(t, f)
}
