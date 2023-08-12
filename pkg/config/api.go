package config

import (
	"github.com/spf13/viper"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
)

func APIPort() uint16 {
	return uint16(viper.GetInt(types.NodeAPIPort))
}

func APIHost() string {
	return viper.GetString(types.NodeAPIHost)
}
