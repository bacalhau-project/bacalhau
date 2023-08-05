package config_v2

import (
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"

	"github.com/bacalhau-project/bacalhau/pkg/logger"
)

func GetLogMode() logger.LogMode {
	val := viper.GetString(NodeLoggingMode)
	out, err := logger.ParseLogMode(val)
	if err != nil {
		log.Warn().Err(err).Msgf("invalid logging mode specified: %s", val)
	}
	return out
}
