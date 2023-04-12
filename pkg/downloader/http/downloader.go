package http

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/bacalhau-project/bacalhau/pkg/util/closer"
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

func (httpDownloader *Downloader) DescribeResult(ctx context.Context, result model.PublishedResult) (map[string]string, error) {
	return nil, errors.New("not implemented for httpdownloader")
}

func (httpDownloader *Downloader) FetchResult(ctx context.Context, item model.DownloadItem) error {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), "pkg/downloader/http.Downloader.FetchResults")
	defer span.End()

	err := func() error {
		log.Ctx(ctx).Debug().Msgf(
			"Downloading result URL %s '%s' to '%s'...",
			item.Name,
			item.CID, item.Target,
		)

		innerCtx, cancel := context.WithDeadline(ctx, time.Now().Add(httpDownloader.Settings.Timeout))
		defer cancel()

		return fetch(innerCtx, item.CID, item.Target)
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
