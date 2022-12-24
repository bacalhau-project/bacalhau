package downloader

import "time"

const (
	DownloadVolumesFolderName = "combined_results"
	DownloadShardsFolderName  = "per_shard"
	DownloadCIDsFolderName    = "raw"
	DownloadFilenameStdout    = "stdout"
	DownloadFilenameStderr    = "stderr"
	DownloadFilenameExitCode  = "exitCode"
	DownloadFolderPerm        = 0755
	DownloadFilePerm          = 0644
	DefaultIPFSTimeout        = 5 * time.Minute
)
