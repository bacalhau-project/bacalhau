package main

import (
	"os"

	"github.com/bacalhau-project/bacalhau/cmd/cli"
	"github.com/bacalhau-project/bacalhau/pkg/config"
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

	/*
		if err := system.InitConfig(); err != nil {
			fmt.Fprintf(os.Stderr, "Failed to initialize config: %s\n", err)
			os.Exit(1)
		}
	*/

	cli.Execute()
}
