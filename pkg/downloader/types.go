package downloader

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/model"
)

type Downloader interface {
	model.Providable

	// FetchResult fetches published result and saves it to downloadPath
	FetchResult(ctx context.Context, result model.PublishedResult, downloadPath string) error
}

type DownloaderProvider interface {
	model.Provider[model.StorageSourceType, Downloader]
}
