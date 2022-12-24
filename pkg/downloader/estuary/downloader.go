package estuary

import (
	"context"
	"fmt"
	"github.com/filecoin-project/bacalhau/pkg/downloader"
	"github.com/filecoin-project/bacalhau/pkg/downloader/http"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"path/filepath"
)

const DefaultEstuaryDownloadGateway string = "https://api.estuary.tech/gw/ipfs"

// EstuaryDownloader uses HTTPDownloader to download result published to Estuary
// by combining Estuary gateway URL and CID returned by Estuary publisher and passing it for download.
type EstuaryDownloader struct {
	*http.HTTPDownloader
	Settings *downloader.DownloadSettings
}

func NewEstuaryDownloader(settings *downloader.DownloadSettings) (*EstuaryDownloader, error) {
	httpDownloader, err := http.NewHTTPDownloader(settings)
	if err != nil {
		return nil, err
	}

	return &EstuaryDownloader{
		HTTPDownloader: httpDownloader,
		Settings:       settings,
	}, nil
}

func (estuaryDownloader *EstuaryDownloader) GetResultsOutputDir() (string, error) {
	return filepath.Abs(estuaryDownloader.Settings.OutputDir)
}

func (estuaryDownloader *EstuaryDownloader) FetchResult(ctx context.Context, shardCIDContext downloader.ShardCIDContext) error {
	ctx, span := system.GetTracer().Start(ctx, "pkg/estuaryDownloader.estuary.FetchResult")
	defer span.End()

	url := fmt.Sprintf("%s/%s", DefaultEstuaryDownloadGateway, shardCIDContext.Result.Data.CID)
	shardCIDContext.Result.Data.URL = url

	return estuaryDownloader.HTTPDownloader.FetchResult(ctx, shardCIDContext)
}
