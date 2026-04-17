package setup

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/repo"
	"github.com/bacalhau-project/bacalhau/pkg/repo/migrations"
)

func SetupMigrationManager() (*repo.MigrationManager, error) {
	return repo.NewMigrationManager(
		repo.NewMigration(repo.Version4, repo.Version5, migrations.V4ToV5),
	)
}

// SetupBacalhauRepo ensures that a bacalhau repo and config exist and are initialized.
func SetupBacalhauRepo(cfg types.Bacalhau) (*repo.FsRepo, error) {
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

func SetupBacalhauRepoForTesting(t testing.TB) (*repo.FsRepo, types.Bacalhau) {
	// create a temporary dir to serve as bacalhau repo whose name includes the current time to avoid collisions with
	/// other tests
	tmpDir, err := os.MkdirTemp("", "")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(tmpDir, fmt.Sprint(time.Now().UnixNano()))

	// disable update checks in testing.
	t.Setenv(config.KeyAsEnvVar(types.UpdateConfigIntervalKey), "0")
	// don't send analytics data during testing
	t.Setenv(config.KeyAsEnvVar(types.DisableAnalyticsKey), "true")
	cfgValues := map[string]any{
		types.DataDirKey: path,
		// callers of this method currently assume it creates an orchestrator node.
		types.OrchestratorEnabledKey: true,
	}

	// the BACALHAU_IPFS_CONNECT env var is only bound if it's corresponding flags are registered.
	// This is because viper cannot bind its value to any existing keys via viper.AutomaticEnv() since the
	// environment variable doesn't map to a key in the config, so we add special handling here until we move away
	// from this flag to the dedicated flags like "BACALHAU_PUBLISHER_IPFS_ENDPOINT",
	// "BACALHAU_INPUTSOURCES_IPFS_ENDPOINT", etc which have a direct mapping to the config key based on their name.
	ipfsConnect := os.Getenv("BACALHAU_IPFS_CONNECT")
	if ipfsConnect == "" {
		ipfsConnect = os.Getenv("BACALHAU_NODE_IPFS_CONNECT")
	}
	if ipfsConnect != "" {
		cfgValues[types.PublishersTypesIPFSEndpointKey] = ipfsConnect
		cfgValues[types.ResultDownloadersTypesIPFSEndpointKey] = ipfsConnect
		cfgValues[types.InputSourcesTypesIPFSEndpointKey] = ipfsConnect
	}

	// init a config with this viper instance using the local configuration as default
	c, err := config.New(config.WithValues(cfgValues))
	if err != nil {
		t.Fatal(err)
	}

	t.Logf("creating repo for testing at: %s", path)
	t.Cleanup(func() {
		// This may fail to clean up due to nats store, log an error, don't fail testing
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Logf("failed to clean up repo at: %s: %s", path, err)
		}
	})
	var cfg types.Bacalhau
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
