package estuary

import (
	"context"

	"github.com/filecoin-project/bacalhau/pkg/downloader/ipfs"
	"go.uber.org/multierr"

	"github.com/filecoin-project/bacalhau/pkg/downloader/http"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/system"
)

type DownloaderParams struct {
	HTTPDownloader *http.Downloader
	IPFSDownloader *ipfs.Downloader
}

// Estuary downloader uses HTTP downloader to download result published to Estuary
// by combining Estuary gateway URL and CID returned by Estuary publisher and passing it for download.
type Downloader struct {
	httpDownloader *http.Downloader
	ipfsDownloader *ipfs.Downloader
}

func NewEstuaryDownloader(params DownloaderParams) *Downloader {
	return &Downloader{
		ipfsDownloader: params.IPFSDownloader,
		httpDownloader: params.HTTPDownloader,
	}
}

func (downloader *Downloader) IsInstalled(ctx context.Context) (bool, error) {
	ipfsInstalled, ipfsErr := downloader.ipfsDownloader.IsInstalled(ctx)
	httpInstalled, httpErr := downloader.httpDownloader.IsInstalled(ctx)
	return ipfsInstalled && httpInstalled, multierr.Combine(ipfsErr, httpErr)
}

func (downloader *Downloader) FetchResult(ctx context.Context, result model.PublishedResult, downloadPath string) error {
	ctx, span := system.GetTracer().Start(ctx, "pkg/downloader.estuary.FetchResult")
	defer span.End()

	// fallback to ipfs download for old results without URL
	if result.Data.URL == "" {
		return downloader.ipfsDownloader.FetchResult(ctx, result, downloadPath)
	}

	return downloader.httpDownloader.FetchResult(ctx, result, downloadPath)
}
