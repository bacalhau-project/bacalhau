package model

const (
	DownloadFilenameStdout   = "stdout"
	DownloadFilenameStderr   = "stderr"
	DownloadFilenameExitCode = "exitCode"
)

type DownloaderSettings struct {
	TimeoutSecs    int
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
