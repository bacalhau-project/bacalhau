package model

import "time"

const (
	DownloadFilenameStdout   = "stdout"
	DownloadFilenameStderr   = "stderr"
	DownloadFilenameExitCode = "exitCode"
)

type DownloaderSettings struct {
	TimeoutSecs    time.Duration
	OutputDir      string
	IPFSSwarmAddrs string
}

type PublishedShardDownloadContext struct {
	Result         PublishedResult
	OutputVolumes  []StorageSpec
	RootDir        string
	CIDDownloadDir string
	ShardDir       string
	VolumeDir      string
}
