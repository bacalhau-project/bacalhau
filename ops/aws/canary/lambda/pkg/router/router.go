package router

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/ops/aws/canary/pkg/models"
	"github.com/bacalhau-project/bacalhau/ops/aws/canary/pkg/scenarios"
	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/repo"
)

var TestcasesMap = map[string]Handler{
	"list":                      scenarios.List,
	"submit":                    scenarios.Submit,
	"submitAndGet":              scenarios.SubmitAndGet,
	"submitDockerIPFSJobAndGet": scenarios.SubmitDockerIPFSJobAndGet,
	"submitAndDescribe":         scenarios.SubmitAnDescribe,
	"submitWithConcurrency":     scenarios.SubmitWithConcurrency,
}

func init() {
	// init system configs
	// set the default configuration
	if err := config.SetViperDefaults(config.Default); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to set up default config values: %s\n", err)
		os.Exit(1)
	}

	if err := setupBacalhauRepo(); err != nil {
		fmt.Fprintf(os.Stderr, "Faild to initalize bacalhau repo: %s", err)
		os.Exit(1)
	}
}

func Route(ctx context.Context, event models.Event) error {
	handler, ok := TestcasesMap[event.Action]
	if !ok {
		return fmt.Errorf("no handler found for action: %s", event.Action)
	}
	err := handler(ctx)
	if err != nil {
		return fmt.Errorf("testcase %s failed: %s", event.Action, err)
	}
	log.Info().Msgf("testcase %s passed", event.Action)
	return nil
}

func setupBacalhauRepo() error {
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
			return fmt.Errorf("failed to get user home dir: %w", err)
		}
		configDir = filepath.Join(home, ".bacalhau")
	}
	fsRepo, err := repo.NewFS(configDir)
	if err != nil {
		return fmt.Errorf("failed to create repo: %w", err)
	}
	if err := fsRepo.Init(); err != nil {
		return fmt.Errorf("failed to initalize repo: %w", err)
	}
	return nil
}
