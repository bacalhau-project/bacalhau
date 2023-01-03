package model

import "time"

const (
	DownloadFilenameStdout   = "stdout"
	DownloadFilenameStderr   = "stderr"
	DownloadFilenameExitCode = "exitCode"
)

type DownloaderSettings struct {
	Timeout        time.Duration
	OutputDir      string
	IPFSSwarmAddrs string
}
