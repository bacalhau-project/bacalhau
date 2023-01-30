package downloader

import (
	"context"

	"github.com/filecoin-project/bacalhau/pkg/model"
)

type Downloader interface {
	model.Providable

	// FetchResult fetches published result and saves it to downloadPath
	FetchResult(ctx context.Context, result model.PublishedResult, downloadPath string) error
}

type DownloaderProvider interface {
	model.Provider[model.StorageSourceType, Downloader]
}

type shardCIDContext struct {
	Result         model.PublishedResult
	OutputVolumes  []model.StorageSpec
	RootDir        string
	CIDDownloadDir string
	ShardDir       string
	VolumeDir      string
}
