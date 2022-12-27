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

func NewHTTPDownloader(settings *model.DownloaderSettings) (*Downloader, error) {
	return &Downloader{
		Settings: settings,
	}, nil
}

func (httpDownloader *Downloader) FetchResult(ctx context.Context, shardCIDContext model.PublishedShardDownloadContext) error {
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
	_, span := system.GetTracer().Start(ctx, "pkg/downloader.http.fetchHttp")
	defer span.End()
	// Create a new file at the specified filepath
	out, err := os.Create(filepath)
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
