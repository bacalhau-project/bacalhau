package config

import (
	"os"
	"time"

	"github.com/spf13/viper"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
)

// TODO idk where this goes yet these are mostly random

func GetDownloadURLRequestRetries() int {
	return viper.GetInt(types.NodeDownloadURLRequestRetries)
}

func GetDownloadURLRequestTimeout() time.Duration {
	return viper.GetDuration(types.NodeDownloadURLRequestTimeout)
}

func SetVolumeSizeRequestTimeout(value time.Duration) {
	viper.Set(types.NodeVolumeSizeRequestTimeout, value)
}

func GetVolumeSizeRequestTimeout() time.Duration {
	return viper.GetDuration(types.NodeVolumeSizeRequestTimeout)
}

func GetStoragePath() string {
	// TODO make this use the config when I get tests passing
	storagePath := os.Getenv("BACALHAU_STORAGE_PATH")
	if storagePath == "" {
		return os.TempDir()
	}
	return storagePath
}
