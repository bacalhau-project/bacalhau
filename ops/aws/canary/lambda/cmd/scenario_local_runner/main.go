package main

import (
	"context"

	"github.com/bacalhau-project/bacalhau/ops/aws/canary/pkg/models"
	"github.com/bacalhau-project/bacalhau/ops/aws/canary/pkg/router"
	"github.com/rs/zerolog/log"
	flag "github.com/spf13/pflag"
)

func main() {
	var action string
	flag.StringVar(&action, "action", "",
		"Action to test. Useful when testing locally before pushing to lambda")
	flag.Parse()

	log.Info().Msgf("Testing locally the action: %s", action)
	err := router.Route(context.Background(), models.Event{Action: action})
	if err != nil {
		log.Error().Msg(err.Error())
	}
}
