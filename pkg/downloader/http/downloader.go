package http

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/util/closer"
	"github.com/rs/zerolog/log"
)

type Downloader struct {
	Settings *model.DownloaderSettings
}

func NewHTTPDownloader(settings *model.DownloaderSettings) *Downloader {
	return &Downloader{
		Settings: settings,
	}
}

func (httpDownloader *Downloader) IsInstalled(context.Context) (bool, error) {
	return true, nil
}

func (httpDownloader *Downloader) FetchResult(ctx context.Context, result model.PublishedResult, downloadPath string) error {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/downloader/http.Downloader.FetchResults")
	defer span.End()

	err := func() error {
		log.Ctx(ctx).Debug().Msgf(
			"Downloading result URL %s '%s' to '%s'...",
			result.Data.Name,
			result.Data.URL, downloadPath,
		)

		innerCtx, cancel := context.WithDeadline(ctx, time.Now().Add(httpDownloader.Settings.Timeout))
		defer cancel()

		return fetch(innerCtx, result.Data.URL, downloadPath)
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
	// Create a new file at the specified filepath
	out, err := os.OpenFile(filepath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, model.DownloadFilePerm)
	if err != nil {
		return err
	}
	defer closer.CloseWithLogOnError("file", out)

	// Make an HTTP GET request to the URL
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	response, err := http.DefaultClient.Do(req) //nolint
	if err != nil {
		return err
	}

	defer closer.DrainAndCloseWithLogOnError(ctx, "http response", response.Body)

	// Write the contents of the response body to the file
	_, err = io.Copy(out, response.Body)
	if err != nil {
		return err
	}

	return nil
}
