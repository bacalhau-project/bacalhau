package downloader

import "context"

// SpecialFiles - i.e. aything that is not a volume
// the boolean value is whether we should append to the global log
var SpecialFiles = map[string]bool{
	DownloadFilenameStdout:   true,
	DownloadFilenameStderr:   true,
	DownloadFilenameExitCode: false,
}

type DownloadSettings struct {
	TimeoutSecs    int
	OutputDir      string
	IPFSSwarmAddrs string
}

type Downloader interface {
	// GetResultsOutputDir returns output dir given in DownloadSettings
	GetResultsOutputDir() (string, error)
	// FetchResults ...
	FetchResults(ctx context.Context, shardCidContext shardCIDContext) error
}
