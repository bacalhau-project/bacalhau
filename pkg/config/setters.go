package config

import (
	"github.com/spf13/viper"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
)

func SetUserKey(path string) {
	viper.Set(types.UserKeyPath, path)
}

func SetLibp2pKey(path string) {
	viper.Set(types.UserLibp2pKeyPath, path)
}

func SetExecutorPluginPath(path string) {
	viper.Set(types.NodeExecutorPluginPath, path)
}

func SetComputeStoragesPath(path string) {
	viper.Set(types.NodeComputeStoragePath, path)
}

func SetAutoCertCachePath(path string) {
	viper.Set(types.NodeServerAPITLSAutoCertCachePath, path)
}
