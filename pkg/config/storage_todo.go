package config

import (
	"os"
	"time"

	"github.com/spf13/viper"
)

// TODO idk where this goes yet these are mostly random

func GetDownloadURLRequestRetries() int {
	return viper.GetInt(NodeDownloadURLRequestRetries)
}

func GetDownloadURLRequestTimeout() time.Duration {
	return viper.GetDuration(NodeDownloadURLRequestTimeout)
}

func SetVolumeSizeRequestTimeout(value time.Duration) {
	viper.Set(NodeVolumeSizeRequestTimeout, value)
}

func GetVolumeSizeRequestTimeout() time.Duration {
	return viper.GetDuration(NodeVolumeSizeRequestTimeout)
}

func GetStoragePath() string {
	// TODO make this use the config when I get tests passing
	storagePath := os.Getenv("BACALHAU_STORAGE_PATH")
	if storagePath == "" {
		return os.TempDir()
	}
	return storagePath
}
