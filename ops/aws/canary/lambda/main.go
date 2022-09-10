package main

import (
	"context"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/rs/zerolog/log"
	flag "github.com/spf13/pflag"
)

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
