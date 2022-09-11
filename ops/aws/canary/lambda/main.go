package main

import (
	"context"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	flag "github.com/spf13/pflag"
	"os"
)

func init() {
	// init system configs
	err := system.InitConfig()
	if err != nil {
		panic(err)
	}

	// override bacalhau logger to print better on cloudwatch logs by removing colors and timestamp
	log.Logger = zerolog.New(zerolog.ConsoleWriter{
		Out:     os.Stderr,
		NoColor: true,
		PartsExclude: []string{
			zerolog.TimestampFieldName,
		},
	})
}

func main() {
	var action string
	flag.StringVar(&action, "action", "",
		"Action to test. Useful when testing locally before pushing to lambda")
	flag.Parse()

	if action != "" {
		log.Info().Msgf("Testing locally the action: %s", action)
		err := route(context.Background(), Event{Action: action})
		if err != nil {
			log.Error().Msg(err.Error())
		}
	} else {
		// running in lambda
		lambda.Start(route)
	}
}
