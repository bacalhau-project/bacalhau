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
	path := viper.GetString(NodeComputeStoragePath)
	// TODO this is left over from previous behaviour
	if path == "" {
		return os.TempDir()
	} else {
		return path
	}
}
