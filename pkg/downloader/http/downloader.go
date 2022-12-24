package http

import (
	"context"
	"errors"
	"github.com/filecoin-project/bacalhau/pkg/downloader"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/rs/zerolog/log"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type HTTPDownloader struct {
	Settings *downloader.DownloadSettings
}

func NewHTTPDownloader(settings *downloader.DownloadSettings) (*HTTPDownloader, error) {
	return &HTTPDownloader{
		Settings: settings,
	}, nil
}

func (httpDownloader *HTTPDownloader) GetResultsOutputDir() (string, error) {
	return filepath.Abs(httpDownloader.Settings.OutputDir)
}

func (httpDownloader *HTTPDownloader) FetchResult(ctx context.Context, shardCIDContext downloader.ShardCIDContext) error {
	ctx, span := system.GetTracer().Start(ctx, "pkg/httpDownloader.http.FetchResult")
	defer span.End()

	err := func() error {
		log.Ctx(ctx).Debug().Msgf(
			"Downloading result CID %s '%s' to '%s'...",
			shardCIDContext.Result.Data.Name,
			shardCIDContext.Result.Data.CID, shardCIDContext.CIDDownloadDir,
		)

		innerCtx, cancel := context.WithDeadline(ctx,
			time.Now().Add(time.Second*time.Duration(httpDownloader.Settings.TimeoutSecs)))
		defer cancel()

		return fetch(innerCtx, shardCIDContext.Result.Data.URL, shardCIDContext.CIDDownloadDir)
	}()

	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			log.Ctx(ctx).Error().Msg("Timed out while downloading result.")
		}

		return err
	}
	return nil
}

func fetch(ctx context.Context, url string, filepath string) error {
	ctx, span := system.GetTracer().Start(ctx, "pkg/downloader.http.fetchHttp")
	defer span.End()
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
