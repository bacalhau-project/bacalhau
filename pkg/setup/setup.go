package setup

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"

	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/repo"
)

// SetupBacalhauRepo ensures that a bacalhau repo and config exist and are initalized.
func SetupBacalhauRepo() (string, error) {
	// set the default configuration
	if err := config.SetViperDefaults(types.Default); err != nil {
		return "", fmt.Errorf("fialed to set up default config values: %w", err)
	}
	configDir := os.Getenv("BACALHAU_DIR")
	//If FIL_WALLET_ADDRESS is set, assumes that ROOT_DIR is the config dir for Station
	//and not a generic environment variable set by the user
	if _, set := os.LookupEnv("FIL_WALLET_ADDRESS"); configDir == "" && set {
		configDir = os.Getenv("ROOT_DIR")
	}
	log.Debug().Msg("BACALHAU_DIR not set, using default of ~/.bacalhau")

	if configDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get user home dir: %w", err)
		}
		configDir = filepath.Join(home, ".bacalhau")
	}
	fsRepo, err := setupRepo(configDir)
	if err != nil {
		return "", err
	}
	return fsRepo.Path()
}

func setupRepo(path string) (*repo.FsRepo, error) {
	fsRepo, err := repo.NewFS(path)
	if err != nil {
		return nil, fmt.Errorf("failed to create repo: %w", err)
	}
	if err := fsRepo.Init(); err != nil {
		return nil, fmt.Errorf("failed to initalize repo: %w", err)
	}
	return fsRepo, nil

}

func SetupBacalhauRepoForTesting(t testing.TB) *repo.FsRepo {
	viper.Reset()
	// TODO pass a testing config
	// set the default configuration
	if err := config.SetViperDefaults(types.Default); err != nil {
		t.Fatal(fmt.Sprintf("fialed to set up default config values: %s", err))
	}

	path := filepath.Join(t.TempDir(), t.Name())
	t.Logf("creating repo for testing at: %s", path)
	fsRepo, err := setupRepo(path)
	if err != nil {
		t.Fatal(err)
	}
	return fsRepo
}
