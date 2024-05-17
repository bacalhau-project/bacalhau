package setup

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/viper"

	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/configenv"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/repo/migrations"

	"github.com/bacalhau-project/bacalhau/pkg/repo"
)

func SetupMigrationManager() (*repo.MigrationManager, error) {
	return repo.NewMigrationManager(
		migrations.V1Migration,
		migrations.V2Migration,
	)
}

// SetupBacalhauRepo ensures that a bacalhau repo and config exist and are initialized.
func SetupBacalhauRepo(repoDir string, c config.ReadWriter) (*repo.FsRepo, error) {
	migrationManger, err := SetupMigrationManager()
	if err != nil {
		return nil, fmt.Errorf("failed to create migration manager: %w", err)
	}
	fsRepo, err := repo.NewFS(repo.FsRepoParams{
		Path:       repoDir,
		Migrations: migrationManger,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create repo: %w", err)
	}
	if exists, err := fsRepo.Exists(); err != nil {
		if repo.IsUnknownVersion(err) {
			return nil, err
		}

		return nil, fmt.Errorf("failed to check if repo exists: %w", err)
	} else if !exists {
		if err := fsRepo.Init(c); err != nil {
			return nil, fmt.Errorf("failed to initialize repo: %w", err)
		}
	} else {
		if err := fsRepo.Open(c); err != nil {
			return nil, fmt.Errorf("failed to open repo: %w", err)
		}
	}
	return fsRepo, nil
}

func SetupBacalhauRepoForTesting(t testing.TB) (*repo.FsRepo, types.BacalhauConfig) {
	// create a specific viper instance for testing
	v := viper.New()
	// init a config with this viper instance using the local configuration as default
	cfg := config.New(config.WithDefaultConfig(configenv.Local), config.WithViper(v))

	// create a temporary dir to serve as bacalhau repo whose name includes the current time to avoid collisions with
	/// other tests
	tmpDir, err := os.MkdirTemp("", "")
	if err != nil {
		t.Fatal(errors.Wrap(err, "failed to create temporary directory in test setup"))
	}
	path := filepath.Join(tmpDir, fmt.Sprint(time.Now().UnixNano()))
	// configure the repo on the testing viper insance
	v.Set("repo", path)

	t.Logf("creating repo for testing at: %s", path)
	t.Setenv("BACALHAU_ENVIRONMENT", "local")
	t.Setenv("BACALHAU_DIR", path)

	// create the repo used for testing
	fsRepo, err := SetupBacalhauRepo(path, cfg)
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		// This may fail on windows, and if so then we'll log the error but not fail the test.
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Logf("failed to clean up repo at: %s: %s", path, err)
		}
	})

	// load the config with merged in settings from repo.
	bcfg, err := cfg.Current()
	if err != nil {
		t.Fatal(err)
	}

	return fsRepo, bcfg
}
