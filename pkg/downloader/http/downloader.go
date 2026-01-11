package http

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/downloader"
	"github.com/bacalhau-project/bacalhau/pkg/storage/url/urldownload"
	"github.com/bacalhau-project/bacalhau/pkg/util/closer"
)

// Replace slashes with some other character that is valid for filenames in most operating systems
var urlSanitizer = strings.NewReplacer("/", "_", "\\", "_", ":", "_", "*", "_", "?", "_", "\"", "_", "<", "_", ">", "_", "|", "_")

type Downloader struct {
	httpClient *http.Client
}

// NewHTTPDownloader creates a new HTTPDownloader with the given settings.
func NewHTTPDownloader() *Downloader {
	return &Downloader{httpClient: http.DefaultClient}
}

// IsInstalled checks if the downloader is ready to be used.
func (httpDownloader *Downloader) IsInstalled(ctx context.Context) (bool, error) {
	// For HTTPDownloader, we can always return true as there's no installation needed.
	return true, nil
}

// FetchResult downloads the result of a computation and saves it to a local file.
func (httpDownloader *Downloader) FetchResult(ctx context.Context, item downloader.DownloadItem) (string, error) {
	sourceSpec, err := urldownload.DecodeSpec(item.Result)
	if err != nil {
		return "", err
	}

	// Get the path and sanitize it for use as a flat filename
	flatFileName, err := SanitizeFileName(sourceSpec.URL)
	if err != nil {
		return "", err
	}

	// Full path to the file
	localPath := filepath.Join(item.ParentPath, flatFileName)
	alreadyExists, err := downloader.IsAlreadyDownloaded(localPath)
	if err != nil {
		return "", err
	}
	if alreadyExists {
		log.Ctx(ctx).Debug().
			Str("URL", sourceSpec.URL).
			Msg("File already downloaded.")
		return localPath, nil
	}

	return localPath, httpDownloader.fetch(ctx, sourceSpec.URL, localPath)
}

// fetch makes an HTTP GET request to the given URL and writes the response to the given filepath.
func (httpDownloader *Downloader) fetch(ctx context.Context, url string, filepath string) error {
	out, err := os.OpenFile(filepath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, downloader.DownloadFilePerm)
	if err != nil {
		return err
	}
	defer closer.CloseWithLogOnError("file", out)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}

	response, err := httpDownloader.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer closer.DrainAndCloseWithLogOnError(ctx, "http response", response.Body)

	if err = checkHTTPResponse(response, url); err != nil {
		return err
	}

	_, err = io.Copy(out, response.Body)
	if err != nil {
		return err
	}

	return nil
}

func checkHTTPResponse(resp *http.Response, url string) error {
	// TODO: Add support for redirects
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	// Read the response body for additional context.
	// Limit the size of the body we will read to avoid large allocations.
	bodyBytes, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20)) //nolint:mnd // 1MB max
	if err != nil {
		// If we can't read the body, just return an error with the status code
		return fmt.Errorf("request to %s failed with status code %d and unable to read response body", url, resp.StatusCode)
	}

	// Close the body before returning the error
	_ = resp.Body.Close()

	// Return an error with the status code and the body content for context
	return fmt.Errorf("request to %s failed with status code %d: %s", url, resp.StatusCode, string(bodyBytes))
}

// SanitizeFileName creates a flat filename from a URL path.
func SanitizeFileName(fullURL string) (string, error) {
	// Parse the URL
	parsedURL, err := url.Parse(fullURL)
	if err != nil {
		return "", err
	}

	urlPath := parsedURL.Host + parsedURL.Path
	return urlSanitizer.Replace(urlPath), nil
}
