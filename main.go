package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"path/filepath"

	"github.com/bacalhau-project/bacalhau/cmd/cli"
	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config_v2"
	"github.com/bacalhau-project/bacalhau/pkg/repo"
	_ "github.com/bacalhau-project/bacalhau/pkg/version"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/logger"
)

func main() {
	defer func() {
		// Make sure any buffered logs are written if something failed before logging was configured.
		logger.LogBufferedLogs(nil)
	}()

	_ = godotenv.Load()

	devstackEnvFile := config.DevstackEnvFile()
	if _, err := os.Stat(devstackEnvFile); err == nil {
		log.Debug().Msgf("Loading environment from %s", devstackEnvFile)
		_ = godotenv.Overload(devstackEnvFile)
	}

	// Ensure commands are able to stop cleanly if someone presses ctrl+c
	ctx, cancel := signal.NotifyContext(context.Background(), util.ShutdownSignals...)
	defer cancel()

	// set the default configuration
	if err := config_v2.SetViperDefaults(config_v2.Default); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to set up default config values: %s\n", err)
		os.Exit(1)
	}

	if err := setupBacalhauRepo(); err != nil {
		fmt.Fprintf(os.Stderr, "Faild to initalize bacalhau repo: %s", err)
		os.Exit(1)
	}

	cli.Execute(ctx)
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
