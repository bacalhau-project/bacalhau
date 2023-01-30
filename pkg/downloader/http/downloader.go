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
	ctx, span := system.GetTracer().Start(ctx, "pkg/httpDownloader.http.FetchResult")
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
	_, span := system.GetTracer().Start(ctx, "pkg/downloader.http.fetchHttp")
	defer span.End()
	// Create a new file at the specified filepath
	out, err := os.OpenFile(filepath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, model.DownloadFilePerm)
	if err != nil {
		return err
	}
	defer out.Close()

	// Make an HTTP GET request to the URL
	response, err := http.Get(url) //nolint
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
