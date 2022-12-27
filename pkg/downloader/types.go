package downloader

import (
	"context"

	"github.com/filecoin-project/bacalhau/pkg/model"
)

type Downloader interface {
	// FetchResult fetches result contained in PublishedShardDownloadContext
	FetchResult(ctx context.Context, shardCidContext model.PublishedShardDownloadContext) error
}

type DownloaderProvider interface {
	GetDownloader(storageType model.StorageSourceType) (Downloader, error)
}
