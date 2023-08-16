package setup

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"

	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config/configenv"
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/repo"
)

type Environment string

const (
	// Known environments that are configured in ops/terraform:
	EnvironmentStaging Environment = "staging"
	EnvironmentProd    Environment = "production"
	EnvironmentDev     Environment = "development"
	EnvironmentTest    Environment = "test"
	EnvironmentLocal   Environment = "local"
)

func (e Environment) String() string {
	return string(e)
}

func (e Environment) IsKnown() bool {
	switch e {
	case EnvironmentStaging, EnvironmentProd, EnvironmentDev, EnvironmentTest, EnvironmentLocal:
		return true
	}
	return false
}

func getEnvironment() Environment {
	env := Environment(os.Getenv("BACALHAU_ENVIRONMENT"))
	if !env.IsKnown() {
		// Log as trace since we don't want to spam CLI users:
		log.Trace().Msgf("BACALHAU_ENVIRONMENT is not set to a known value: %s", env)

		// This usually happens in the case of a short-lived test cluster, in
		// which case we should default to development. However, we want to
		// avoid using any environment-specific settings for IPFS swarms
		// (which are only configured for production and staging)
		if strings.Contains(os.Args[0], "/_test/") ||
			strings.HasSuffix(os.Args[0], ".test") ||
			flag.Lookup("test.v") != nil ||
			flag.Lookup("test.run") != nil {
			env = EnvironmentTest
		} else {
			log.Debug().Msgf("Defaulting to local environment: \n os.Args: %v", os.Args)
			env = EnvironmentLocal
		}
	}
	return env
}

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

// SetupBacalhauRepo ensures that a bacalhau repo and config exist and are initalized.
func SetupBacalhauRepo(repoDir string) (string, error) {
	env := getEnvironment()

	var bacalhauConfig types.BacalhauConfig
	switch env {
	case EnvironmentProd:
		bacalhauConfig = configenv.Production
	case EnvironmentStaging:
		bacalhauConfig = configenv.Staging
	case EnvironmentDev:
		bacalhauConfig = configenv.Development
	case EnvironmentTest:
		bacalhauConfig = configenv.Testing
	case EnvironmentLocal:
		bacalhauConfig = configenv.Local
	default:
		// this would indicate an error in the above logic
		bacalhauConfig = configenv.Local
	}

	// set the default configuration based on the environment
	if err := config.SetViperDefaults(bacalhauConfig); err != nil {
		return "", fmt.Errorf("fialed to set up default config values: %w", err)
	}
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
	if err := fsRepo.Init(); err != nil {
		return nil, fmt.Errorf("failed to initalize repo: %w", err)
	}
	return fsRepo, nil

}

func SetupBacalhauRepoForTesting(t testing.TB) *repo.FsRepo {
	viper.Reset()
	// TODO pass a testing config
	// set the default configuration
	if err := config.SetViperDefaults(configenv.Local); err != nil {
		t.Fatal(fmt.Sprintf("fialed to set up default config values: %s", err))
	}

	path := filepath.Join(os.TempDir(), fmt.Sprint(time.Now().UnixNano()))
	t.Logf("creating repo for testing at: %s", path)
	fsRepo, err := setupRepo(path)
	if err != nil {
		t.Fatal(err)
	}
	return fsRepo
}
