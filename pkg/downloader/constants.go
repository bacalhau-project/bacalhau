package downloader

import "time"

const (
	DownloadVolumesFolderName = "combined_results"
	DownloadShardsFolderName  = "per_shard"
	DownloadCIDsFolderName    = "raw"
	DownloadFolderPerm        = 0755
	DownloadFilePerm          = 0644
	DefaultIPFSTimeout        = 5 * time.Minute
)
