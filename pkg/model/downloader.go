package model

import "time"

const (
	DownloadFilenameStdout   = "stdout"
	DownloadFilenameStderr   = "stderr"
	DownloadFilenameExitCode = "exitCode"
	DownloadCIDsFolderName   = "raw"
	DownloadFolderPerm       = 0755
	DownloadFilePerm         = 0644
	DefaultIPFSTimeout       = 5 * time.Minute
)

type DownloaderSettings struct {
	Timeout        time.Duration
	OutputDir      string
	IPFSSwarmAddrs string
	SingleFile     string
	LocalIPFS      bool
	Raw            bool
}
