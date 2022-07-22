package config

import (
	"os"
	"time"
)

func IsDebug() bool {
	return os.Getenv("LOG_LEVEL") == "debug"
}

func DevstackGetShouldPrintInfo() bool {
	return os.Getenv("DEVSTACK_PRINT_INFO") != ""
}

func DevstackSetShouldPrintInfo() {
	os.Setenv("DEVSTACK_PRINT_INFO", "1")
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

// by default we wait 2 minutes for the IPFS network to resolve a CID
// tests will override this using config.SetVolumeSizeRequestTimeout(2)
var getVolumeSizeRequestTimeoutSeconds int64 = 120

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

// by default we wait 5 minutes for the IPFS network to download a CID
// tests will override this using config.SetVolumeSizeRequestTimeout(2)
var downloadCidRequestTimeoutSeconds int64 = 300

// how long do we wait for a cid to download
func GetDownloadCidRequestTimeout() time.Duration {
	return time.Duration(downloadCidRequestTimeoutSeconds) * time.Second
}

func SetDownloadCidRequestTimeout(seconds int64) {
	downloadCidRequestTimeoutSeconds = seconds
}

// by default we wait 5 minutes for a URL to download
// tests will override this using config.SetDownloadURLRequestTimeoutSeconds(2)
var downloadURLRequestTimeoutSeconds int64 = 300

// how long do we wait for a URL to download
func GetDownloadURLRequestTimeout() time.Duration {
	return time.Duration(downloadURLRequestTimeoutSeconds) * time.Second
}

func SetDownloadURLRequestTimeoutSeconds(seconds int64) {
	downloadURLRequestTimeoutSeconds = seconds
}
