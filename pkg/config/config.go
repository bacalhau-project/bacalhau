package config

import (
	"os"
)

func IsDebug() bool {
	return os.Getenv("LOG_LEVEL") == "debug"
}

func ShouldKeepStack() bool {
	return os.Getenv("KEEP_STACK") != ""
}

func GetStoragePath() string {
	storagePath := os.Getenv("BACALHAU_STORAGE_PATH")
	if storagePath == "" {
		storagePath = os.TempDir()
	}
	return storagePath
}
