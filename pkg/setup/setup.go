package setup

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"

	"github.com/bacalhau-project/bacalhau/pkg/repo"
)

func getBacalhauRepoPath() (string, error) {
	// BACALHAU_DIR has the highest precedence, if its set, we return it
	repoDir := os.Getenv("BACALHAU_DIR")
	if repoDir != "" {
		log.Debug().Str("repo", repoDir).Msg("using BACALHAU_DIR as bacalhau repo")
		return repoDir, nil
	}
	// next precedence is station configuration

	//If FIL_WALLET_ADDRESS is set, assumes that ROOT_DIR is the config dir for Station
	//and not a generic environment variable set by the user
	if _, set := os.LookupEnv("FIL_WALLET_ADDRESS"); set {
		repoDir = os.Getenv("ROOT_DIR")
		if repoDir != "" {
			log.Debug().Str("repo", repoDir).Msg("using station ROOT_DIR as bacalhau repo")
			return repoDir, nil
		}
	}

	// next is the repo flag

	if repoDir = viper.GetString("repo"); repoDir != "" {
		log.Debug().Str("repo", repoDir).Msg("using --repo flag value as bacalhau repo")
		return repoDir, nil
	}

	// last is the default, the home directory
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home dir for bacalhau repo: %w", err)
	}
	repoDir = filepath.Join(home, ".bacalhau")
	log.Info().Str("repo", repoDir).Msg("using $HOME for bacalhau repo")
	return repoDir, nil
}

// SetupBacalhauRepo ensures that a bacalhau repo and config exist and are initialized.
func SetupBacalhauRepo(repoDir string) (string, error) {
	if repoDir == "" {
		var err error
		repoDir, err = getBacalhauRepoPath()
		if err != nil {
			return "", err
		}
	}

	fsRepo, err := setupRepo(repoDir)
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
