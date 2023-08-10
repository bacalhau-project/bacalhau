package config_v2

import "github.com/spf13/viper"

// TODO ensure these exists or return an error

func GetLibp2pTracerPath() string {
	return viper.GetString(NodeMetricsLibp2pTracerPath)
}

func GetEventTracerPath() string {
	return viper.GetString(NodeMetricsEventTracerPath)
}
