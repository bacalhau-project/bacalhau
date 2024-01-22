package setup

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/spf13/viper"

	"github.com/bacalhau-project/bacalhau/pkg/repo"
)

// SetupBacalhauRepo ensures that a bacalhau repo and config exist and are initialized.
func SetupBacalhauRepo(repoDir string) (*repo.FsRepo, error) {
	fsRepo, err := setupRepo(repoDir)
	if err != nil {
		return nil, err
	}
	return fsRepo, nil
}

func setupRepo(path string) (*repo.FsRepo, error) {
	fsRepo, err := repo.NewFS(path)
	if err != nil {
		return nil, fmt.Errorf("failed to create repo: %w", err)
	}
	if exists, err := fsRepo.Exists(); err != nil {
		return nil, fmt.Errorf("failed to check if repo exists: %w", err)
	} else if !exists {
		if err := fsRepo.Init(); err != nil {
			return nil, fmt.Errorf("failed to initialize repo: %w", err)
		}
	} else {
		if err := fsRepo.Open(); err != nil {
			return nil, fmt.Errorf("failed to open repo: %w", err)
		}
	}
	return fsRepo, nil
}

func SetupBacalhauRepoForTesting(t testing.TB) *repo.FsRepo {
	viper.Reset()

	path := filepath.Join(os.TempDir(), fmt.Sprint(time.Now().UnixNano()))
	t.Logf("creating repo for testing at: %s", path)
	t.Setenv("BACALHAU_ENVIRONMENT", "local")
	t.Setenv("BACALHAU_DIR", path)
	fsRepo, err := setupRepo(path)
	if err != nil {
		t.Fatal(err)
	}
	return fsRepo
}
