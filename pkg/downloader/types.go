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

type DownloadClient interface {
	Get(ctx context.Context, cid string, downloadDir string) error
}

type Downloader interface {
	GetResultsOutputDir() (string, error)
	FetchResults(ctx context.Context, shardCidContext shardCIDContext) error
}
