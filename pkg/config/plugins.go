package config

import (
	"github.com/spf13/viper"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
)

func GetExecutorPluginsPath() string {
	return viper.GetString(types.NodeExecutorPluginPath)
}
