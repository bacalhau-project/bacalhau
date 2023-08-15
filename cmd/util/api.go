package util

import (
	"context"
	"strconv"

	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"

	pubapi "github.com/bacalhau-project/bacalhau/pkg/publicapi"
	"github.com/bacalhau-project/bacalhau/pkg/requester/publicapi"
)

func GetAPIClient(ctx context.Context) *publicapi.RequesterAPIClient {
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

	tlsConfig := &pubapi.ClientTLSConfig{}
	if cert := viper.GetString("cacert"); cert != "" {
		tlsConfig.CACert = cert
	} else {
		tlsConfig.AllowInsecure = viper.GetBool("insecure")
	}

	return publicapi.NewRequesterAPIClient(apiHost, apiPort, tlsConfig)
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
