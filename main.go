package main

import (
	"time"

	"github.com/filecoin-project/bacalhau/cmd/bacalhau"
	"github.com/filecoin-project/bacalhau/internal/logger"
	"github.com/rs/zerolog/log"
)

func main() {
	logger.Initialize()

	start := time.Now()
	log.Trace().Msgf("Top of execution - %s", start.UTC())
	bacalhau.Execute()
	log.Trace().Msgf("Execution finished - %s", time.Since(start))
}
