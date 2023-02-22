package main

import (
	"os"

	"github.com/filecoin-project/bacalhau/pkg/config"
	_ "github.com/filecoin-project/bacalhau/pkg/version"

	"github.com/filecoin-project/bacalhau/cmd/bacalhau"

	"github.com/filecoin-project/bacalhau/pkg/logger"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/joho/godotenv"
	"github.com/rs/zerolog/log"
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

	if err := system.InitConfig(); err != nil {
		log.Error().Msgf("Failed to initialize config: %s", err)
		os.Exit(1)
	}

	bacalhau.Execute()
}
