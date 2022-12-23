package downloader

import (
	"context"
	"errors"
	"fmt"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/rs/zerolog/log"
	"io"
	"net/http"
	"os"
	"path/filepath"
)

type httpDownloader struct {
	settings *DownloadSettings
}

func NewHTTPDownloader(ctx context.Context, cm *system.CleanupManager, settings *DownloadSettings) (*httpDownloader, error) {
	return &httpDownloader{
		settings: settings,
	}, nil
}

func (downloader *httpDownloader) GetResultsOutputDir() (string, error) {
	return filepath.Abs(downloader.settings.OutputDir)
}

func (downloader *httpDownloader) FetchResults(ctx context.Context, shardCIDContext shardCIDContext) error {
	ctx, span := system.GetTracer().Start(ctx, "pkg/downloader.http.FetchResults")
	defer span.End()

	err := func() error {
		log.Ctx(ctx).Debug().Msgf(
			"Downloading result CID %s '%s' to '%s'...",
			shardCIDContext.result.Data.Name,
			shardCIDContext.result.Data.CID, shardCIDContext.cidDownloadDir,
		)

		//innerCtx, cancel := context.WithDeadline(ctx,
		//	time.Now().Add(time.Second*time.Duration(downloader.settings.TimeoutSecs)))
		//defer cancel()

		url := fmt.Sprintf("https://api.estuary.tech/gw/ipfs/%s", shardCIDContext.result.Data.CID)

		return fetchHttp(url, shardCIDContext.cidDownloadDir)
	}()

	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			log.Ctx(ctx).Error().Msg("Timed out while downloading result.")
		}

		return err
	}
	return nil
}

func fetchHttp(url string, filepath string) error {
	// Create a new file at the specified filepath
	out, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer out.Close()

	// Make an HTTP GET request to the URL
	response, err := http.Get(url)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	// Write the contents of the response body to the file
	_, err = io.Copy(out, response.Body)
	if err != nil {
		return err
	}

	return nil
}
