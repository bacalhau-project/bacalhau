package estuary

import (
	"context"

	"github.com/filecoin-project/bacalhau/pkg/downloader/http"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/system"
)

// Estuary downloader uses HTTP downloader to download result published to Estuary
// by combining Estuary gateway URL and CID returned by Estuary publisher and passing it for download.
type Downloader struct {
	Settings       *model.DownloaderSettings
	httpDownloader *http.Downloader
}

func NewEstuaryDownloader(settings *model.DownloaderSettings) (*Downloader, error) {
	httpDownloader, err := http.NewHTTPDownloader(settings)
	if err != nil {
		return nil, err
	}

	return &Downloader{
		httpDownloader: httpDownloader,
	}, nil
}

func (downloader *Downloader) FetchResult(ctx context.Context, result model.PublishedResult, downloadDir string) error {
	ctx, span := system.GetTracer().Start(ctx, "pkg/downloader.estuary.FetchResult")
	defer span.End()

	return downloader.httpDownloader.FetchResult(ctx, result, downloadDir)
}
