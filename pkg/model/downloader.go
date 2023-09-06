package model

import "time"

const (
	DownloadFilenameStdout   = "stdout"
	DownloadFilenameStderr   = "stderr"
	DownloadFilenameExitCode = "exitCode"
	DownloadCIDsFolderName   = "raw"
	DownloadFolderPerm       = 0755
	DownloadFilePerm         = 0644
	DefaultDownloadTimeout   = 5 * time.Minute
)

type DownloaderSettings struct {
	Timeout    time.Duration
	OutputDir  string
	SingleFile string
	Raw        bool
}
