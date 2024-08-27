package setup

import (
	"fmt"
	"path/filepath"
	"testing"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/configv2"
	types2 "github.com/bacalhau-project/bacalhau/pkg/configv2/types"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
	"github.com/bacalhau-project/bacalhau/pkg/repo/migrations"

	"github.com/bacalhau-project/bacalhau/pkg/repo"
)

func SetupMigrationManager() (*repo.MigrationManager, error) {
	return repo.NewMigrationManager(
		migrations.V1Migration,
		migrations.V2Migration,
		migrations.V3Migration,
	)
}

// SetupBacalhauRepo ensures that a bacalhau repo and config exist and are initialized.
func SetupBacalhauRepo(cfg types2.Bacalhau) (*repo.FsRepo, error) {
	if err := logger.ConfigureLogging(cfg.Logging.Mode, cfg.Logging.Level); err != nil {
		return nil, fmt.Errorf("failed to configure logging: %w", err)
	}
	migrationManger, err := SetupMigrationManager()
	if err != nil {
		return nil, fmt.Errorf("failed to create migration manager: %w", err)
	}
	fsRepo, err := repo.NewFS(repo.FsRepoParams{
		Path:       cfg.DataDir,
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
		if err := fsRepo.Init(cfg); err != nil {
			return nil, fmt.Errorf("failed to initialize repo: %w", err)
		}
	} else {
		if err := fsRepo.Open(); err != nil {
			return nil, fmt.Errorf("failed to open repo: %w", err)
		}
	}
	return fsRepo, nil
}

func SetupBacalhauRepoForTesting(t testing.TB) (*repo.FsRepo, types2.Bacalhau) {
	// create a temporary dir to serve as bacalhau repo whose name includes the current time to avoid collisions with
	/// other tests
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, fmt.Sprint(time.Now().UnixNano()))

	// disable update checks in testing.
	t.Setenv("BACALHAU_UPDATECONFIG_INTERVAL", "0")

	// configure the repo on the testing viper insance
	// init a config with this viper instance using the local configuration as default
	c, err := configv2.New(configv2.WithValues(map[string]any{
		"DataDir": path,
	}))
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("creating repo for testing at: %s", path)

	var cfg types2.Bacalhau
	if err := c.Unmarshal(&cfg); err != nil {
		t.Fatal(err)
	}
	// create the repo used for testing
	fsRepo, err := SetupBacalhauRepo(cfg)
	if err != nil {
		t.Fatal(err)
	}

	return fsRepo, cfg
}
