package config

import (
	"github.com/rs/zerolog/log"
	"github.com/spf13/viper"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/logger"
)

func GetLogMode() logger.LogMode {
	mode := viper.Get(types.NodeLoggingMode)
	switch v := mode.(type) {
	case logger.LogMode:
		return v
	case string:
		out, err := logger.ParseLogMode(v)
		if err != nil {
			log.Warn().Err(err).Msgf("invalid logging mode specified: %s", v)
		}
		return out
	default:
		log.Error().Msgf("unknown logging mode: %v", mode)
		return logger.LogModeDefault
	}
}
