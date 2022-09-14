package logger

import (
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"os"
)

func SetupCWLogger() {
	// override bacalhau logger to print better on cloudwatch logs by removing colors and timestamp
	log.Logger = zerolog.New(zerolog.ConsoleWriter{
		Out:     os.Stderr,
		NoColor: true,
		PartsExclude: []string{
			zerolog.TimestampFieldName,
		},
	})
}
