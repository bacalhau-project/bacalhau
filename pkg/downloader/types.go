package downloader

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/model"
)

type Downloader interface {
	model.Providable

	// DescribeResult provides information on the contents of the result,
	// providing a mapping between the 'path' of the contents and the
	// identifier used to fetch it (by this Downloader).
	DescribeResult(ctx context.Context, result model.PublishedResult) (map[string]string, error)

	// FetchResult fetches item and saves to disk (as per item's Target)
	FetchResult(ctx context.Context, item model.DownloadItem) error
}

type DownloaderProvider interface {
	model.Provider[model.StorageSourceType, Downloader]
}
