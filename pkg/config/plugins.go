package config

import "github.com/spf13/viper"

func GetExecutorPluginsPath() string {
	return viper.GetString(NodeExecutorPluginPath)
}
