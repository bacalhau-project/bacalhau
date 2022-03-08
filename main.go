package main

import (
	"time"

	"github.com/filecoin-project/bacalhau/cmd/bacalhau"
	"github.com/filecoin-project/bacalhau/internal/logger"
)

func main() {
	start := time.Now()
	logger.Debugf("Top of execution - %s", start.UTC())
	bacalhau.Execute()
	logger.Debugf("Execution finished - %s", time.Since(start))
}
