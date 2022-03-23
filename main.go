package main

import (
	"time"

	"github.com/filecoin-project/bacalhau/cmd/bacalhau"
	_ "github.com/filecoin-project/bacalhau/internal/logger"
	"github.com/rs/zerolog/log"
)

// Values for version are injected by the build.
var (
	VERSION = ""
)

func main() {
	start := time.Now()
	log.Trace().Msgf("Top of execution - %s", start.UTC())
	bacalhau.Execute(VERSION)
	log.Trace().Msgf("Execution finished - %s", time.Since(start))
}
