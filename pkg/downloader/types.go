package downloader

import (
	"context"
	"github.com/filecoin-project/bacalhau/pkg/model"
)

// SpecialFiles - i.e. anything that is not a volume
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

type ShardCIDContext struct {
	Result         model.PublishedResult
	OutputVolumes  []model.StorageSpec
	RootDir        string
	CIDDownloadDir string
	ShardDir       string
	VolumeDir      string
}

type Downloader interface {
	// GetResultsOutputDir returns output dir given in DownloadSettings
	GetResultsOutputDir() (string, error)
	// FetchResult fetches result contained in ShardCIDContext
	FetchResult(ctx context.Context, shardCidContext ShardCIDContext) error
}
