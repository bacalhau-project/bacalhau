package estuary

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/downloader/ipfs"
	"go.uber.org/multierr"

	"github.com/bacalhau-project/bacalhau/pkg/downloader/http"
	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/system"
)

// Estuary downloader uses HTTP downloader to download result published to Estuary
// by combining Estuary gateway URL and CID returned by Estuary publisher and passing it for download.
type Downloader struct {
	httpDownloader *http.Downloader
	ipfsDownloader *ipfs.Downloader
}

func NewEstuaryDownloader(cm *system.CleanupManager, settings *model.DownloaderSettings) *Downloader {
	return &Downloader{
		ipfsDownloader: ipfs.NewIPFSDownloader(cm, settings),
		httpDownloader: http.NewHTTPDownloader(settings),
	}
}

func (downloader *Downloader) IsInstalled(ctx context.Context) (bool, error) {
	ipfsInstalled, ipfsErr := downloader.ipfsDownloader.IsInstalled(ctx)
	httpInstalled, httpErr := downloader.httpDownloader.IsInstalled(ctx)
	return ipfsInstalled && httpInstalled, multierr.Combine(ipfsErr, httpErr)
}

func (downloader *Downloader) DescribeResult(ctx context.Context, result model.PublishedResult) (map[string]string, error) {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/downloader.estuary.FetchResult")
	defer span.End()

	// fallback to ipfs download for old results without URL
	if result.Data.URL == "" {
		return downloader.ipfsDownloader.DescribeResult(ctx, result)
	}

	return downloader.httpDownloader.DescribeResult(ctx, result)
}

func (downloader *Downloader) FetchResult(ctx context.Context, item model.DownloadItem) error {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/downloader.estuary.FetchResult")
	defer span.End()

	if item.CID == "" {
		return downloader.httpDownloader.FetchResult(ctx, item)
	}

	return downloader.ipfsDownloader.FetchResult(ctx, item)
}
