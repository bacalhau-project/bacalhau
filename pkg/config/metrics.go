package config

import (
	"github.com/spf13/viper"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
)

// TODO ensure these exists or return an error

func GetLibp2pTracerPath() string {
	return viper.GetString(types.NodeMetricsLibp2pTracerPath)
}

func GetEventTracerPath() string {
	return viper.GetString(types.NodeMetricsEventTracerPath)
}
