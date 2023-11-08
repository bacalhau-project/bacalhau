package s3signed

import (
	"context"
	"errors"

	"github.com/bacalhau-project/bacalhau/pkg/downloader"
	"github.com/bacalhau-project/bacalhau/pkg/downloader/http"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/s3"
	"github.com/bacalhau-project/bacalhau/pkg/storage/url/urldownload"
)

type DownloaderParams struct {
	HTTPDownloader *http.Downloader
}

type Downloader struct {
	httpDownloader *http.Downloader
}

func NewDownloader(params DownloaderParams) *Downloader {
	return &Downloader{
		httpDownloader: params.HTTPDownloader,
	}
}

func (d *Downloader) IsInstalled(ctx context.Context) (bool, error) {
	return d.httpDownloader.IsInstalled(ctx)
}

func (d *Downloader) FetchResult(ctx context.Context, item downloader.DownloadItem) (string, error) {
	if item.SingleFile != "" {
		return "", errors.New("s3signed downloader does not support single file downloads")
	}

	sourceSpec, err := s3.DecodeSignedResultSpec(item.Result)
	if err != nil {
		return "", err
	}

	urlSourceSpec := &models.SpecConfig{
		Type: models.StorageSourceURL,
		Params: urldownload.Source{
			URL: sourceSpec.SignedURL,
		}.ToMap(),
	}

	return d.httpDownloader.FetchResult(ctx, downloader.DownloadItem{
		Result:     urlSourceSpec,
		SingleFile: item.SingleFile,
		ParentPath: item.ParentPath,
	})
}
