package estuary

import (
	"context"

	"github.com/filecoin-project/bacalhau/pkg/downloader/ipfs"

	"github.com/filecoin-project/bacalhau/pkg/downloader/http"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/system"
)

// Estuary downloader uses HTTP downloader to download result published to Estuary
// by combining Estuary gateway URL and CID returned by Estuary publisher and passing it for download.
type Downloader struct {
	httpDownloader *http.Downloader
	ipfsDownloader *ipfs.Downloader
}

func NewEstuaryDownloader(cm *system.CleanupManager, settings *model.DownloaderSettings) *Downloader {
	return &Downloader{
		httpDownloader: http.NewHTTPDownloader(settings),
		ipfsDownloader: ipfs.NewIPFSDownloader(cm, settings),
	}
}

func (downloader *Downloader) FetchResult(ctx context.Context, result model.PublishedResult, downloadDir string) error {
	ctx, span := system.GetTracer().Start(ctx, "pkg/downloader.estuary.FetchResult")
	defer span.End()

	return downloader.httpDownloader.FetchResult(ctx, result, downloadDir)
}
