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
	return os.Getenv("BACALHAU_STORAGE_PATH")
}
