package util

import (
	"context"
	"strconv"

	"github.com/bacalhau-project/bacalhau/pkg/publicapi/client"
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"
)

func GetAPIClient(ctx context.Context) *client.APIClient {
	var apiHost string
	var apiPort uint16
	if envAPIHost := viper.GetString("api-host"); envAPIHost != "" {
		apiHost = envAPIHost
	}

	if envAPIPort := viper.GetString("api-port"); envAPIPort != "" {
		var parseErr error
		parsedPort, parseErr := strconv.ParseUint(envAPIPort, 10, 16)
		if parseErr != nil {
			log.Ctx(ctx).Fatal().Msgf("could not parse API_PORT into an int. %s", envAPIPort)
		} else {
			apiPort = uint16(parsedPort)
		}
	}

	return client.NewAPIClient(apiHost, apiPort)
}

func GetAPIPort(ctx context.Context) uint16 {
	var apiPort uint16

	if envAPIPort := viper.GetString("api-port"); envAPIPort != "" {
		var parseErr error
		parsedPort, parseErr := strconv.ParseUint(envAPIPort, 10, 16)
		if parseErr != nil {
			log.Ctx(ctx).Fatal().Msgf("could not parse API_PORT into an int. %s", envAPIPort)
		} else {
			apiPort = uint16(parsedPort)
		}
	}
	return apiPort
}
