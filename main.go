package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/bacalhau-project/bacalhau/cmd/cli"
	"github.com/bacalhau-project/bacalhau/cmd/util"
	"github.com/bacalhau-project/bacalhau/pkg/config"
	"github.com/bacalhau-project/bacalhau/pkg/config_v2"
	"github.com/bacalhau-project/bacalhau/pkg/system"
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

	if err := system.InitConfig(); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to initialize config: %s\n", err)
		os.Exit(1)
	}

	cli.Execute(ctx)
}
