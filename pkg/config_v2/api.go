package config_v2

import "github.com/spf13/viper"

func GetAPIPort() uint16 {
	return uint16(viper.GetInt(NodeAPIPort))
}

func GetAPIHost() string {
	return viper.GetString(NodeAPIHost)
}
