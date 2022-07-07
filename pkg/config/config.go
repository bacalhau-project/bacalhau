package config

import (
	"os"
	"time"
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

// by default we wait 10 seconds
var getVolumeSizeRequestTimeoutSeconds int64 = 10

// how long do we wait for a volume size request to timeout
// if a non-existing cid is asked for - the dockerIPFS.IPFSClient.GetCidSize(ctx, volume.Cid)
// function will hang for a long time - so we wrap that call in a timeout
// for tests - we only want to wait for 2 seconds because everything is on a local network
// in prod - we want to wait longer because we might be running a job that is
// using non-local CIDs
// the tests are expected to call SetVolumeSizeRequestTimeout to reduce this timeout
func GetVolumeSizeRequestTimeout() time.Duration {
	return time.Duration(getVolumeSizeRequestTimeoutSeconds) * time.Second
}

func SetVolumeSizeRequestTimeout(seconds int64) {
	getVolumeSizeRequestTimeoutSeconds = seconds
}
