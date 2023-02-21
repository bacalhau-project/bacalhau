package v1beta1

import "time"

const (
	DownloadFilenameStdout    = "stdout"
	DownloadFilenameStderr    = "stderr"
	DownloadFilenameExitCode  = "exitCode"
	DownloadVolumesFolderName = "combined_results"
	DownloadShardsFolderName  = "per_shard"
	DownloadCIDsFolderName    = "raw"
	DownloadFolderPerm        = 0755
	DownloadFilePerm          = 0644
	DefaultIPFSTimeout        = 5 * time.Minute
)

type DownloaderSettings struct {
	Timeout        time.Duration
	OutputDir      string
	IPFSSwarmAddrs string
}
