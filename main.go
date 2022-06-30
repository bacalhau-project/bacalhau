package main

import (
	"os"

	"github.com/filecoin-project/bacalhau/cmd/bacalhau"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/rs/zerolog/log"
)

// Values for version are injected by the build.
var (
	VERSION = ""
)

func main() {
	if err := system.InitConfig(); err != nil {
		log.Error().Msgf("Failed to initialize config: %s", err)
		os.Exit(1)
	}

	bacalhau.Execute(VERSION)
}
