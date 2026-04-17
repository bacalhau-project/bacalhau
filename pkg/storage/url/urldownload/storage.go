package urldownload

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
	"github.com/bacalhau-project/bacalhau/pkg/util/closer"
)

var (
	ErrNoContentLengthFound = errors.New("content-length not provided by the server")
)

// StorageProvider downloads data on request from a URL to a local
// directory.

type StorageProvider struct {
	client *retryablehttp.Client
}

func NewStorage(timeout time.Duration, maxRetries int) *StorageProvider {
	log.Debug().Msg("URL download driver created")

	client := retryablehttp.NewClient()
	client.HTTPClient = &http.Client{
		Timeout: timeout,
		Transport: otelhttp.NewTransport(nil, otelhttp.WithSpanNameFormatter(func(operation string, r *http.Request) string {
			return fmt.Sprintf("%s %s", r.Method, r.URL.Path)
		}), otelhttp.WithSpanOptions(trace.WithAttributes(semconv.PeerService("url-download")))),
	}
	client.RetryMax = maxRetries
	client.RetryWaitMax = time.Second * 1
	client.Logger = retryLogger{}
	client.CheckRetry = func(ctx context.Context, resp *http.Response, err error) (bool, error) {
		if err := ctx.Err(); err != nil {
			return false, err
		}
		if err == nil {
			// Existing behavior around retrying is to retry on _all_ non 2xx status codes. This includes codes that would have no
			// realistic hope of succeeding like `Unauthorized` or `Gone`
			if resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusBadRequest {
				return false, nil
			}
			return true, nil
		}

		return retryablehttp.DefaultRetryPolicy(ctx, resp, err)
	}

	return &StorageProvider{
		client: client,
	}
}

func (sp *StorageProvider) IsInstalled(context.Context) (bool, error) {
	return true, nil
}

func (sp *StorageProvider) HasStorageLocally(context.Context, models.InputSource) (bool, error) {
	return false, nil
}

func (sp *StorageProvider) GetVolumeSize(ctx context.Context, _ *models.Execution, storageSpec models.InputSource) (uint64, error) {
	source, err := DecodeSpec(storageSpec.Source)
	if err != nil {
		return 0, err
	}

	u, err := IsURLSupported(source.URL)
	if err != nil {
		return 0, err
	}

	req, err := retryablehttp.NewRequestWithContext(ctx, http.MethodHead, u.String(), nil)
	if err != nil {
		return 0, err
	}

	res, err := sp.client.Do(req) //nolint:bodyclose // this is being closed - golangci-lint is wrong again
	if err != nil {
		return 0, err
	}
	defer closer.DrainAndCloseWithLogOnError(ctx, "response", res.Body)

	if res.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("received non-OK response code %d while fetching size of file download", res.StatusCode)
	}

	// Ideally if the content size is not provided by server we should try and fetch the file with max size
	// as the one provided in the storageSpec
	if res.ContentLength < 0 {
		return 0, ErrNoContentLengthFound
	}

	return uint64(res.ContentLength), nil
}

// PrepareStorage will download the file from the URL
func (sp *StorageProvider) PrepareStorage(
	ctx context.Context,
	storageDirectory string,
	execution *models.Execution,
	input models.InputSource) (storage.StorageVolume, error) {
	source, err := DecodeSpec(input.Source)
	if err != nil {
		return storage.StorageVolume{}, err
	}
	u, err := IsURLSupported(source.URL)
	if err != nil {
		return storage.StorageVolume{}, err
	}

	// Create a temporary folder inside the provided directory
	outputPath, err := os.MkdirTemp(storageDirectory, "*")
	if err != nil {
		return storage.StorageVolume{}, err
	}

	req, err := retryablehttp.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return storage.StorageVolume{}, err
	}

	requestDidRedirect := false

	// Install handler which can recognize whether we have performed a redirect or not.
	previousRedirect := sp.client.HTTPClient.CheckRedirect
	sp.client.HTTPClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		requestDidRedirect = true
		return nil
	}

	res, err := sp.client.Do(req) //nolint:bodyclose // this is being closed - golangci-lint is wrong again
	if err != nil {
		return storage.StorageVolume{}, fmt.Errorf("failed to begin download from url %s: %w", u, err)
	}
	defer closer.DrainAndCloseWithLogOnError(ctx, "response", res.Body)

	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		return storage.StorageVolume{}, fmt.Errorf("non-200 response from URL (%s): %s", source.URL, res.Status)
	}

	// Reset previous redirect handler
	sp.client.HTTPClient.CheckRedirect = previousRedirect

	var fileName string
	baseName := path.Base(res.Request.URL.Path)

	// Check whether content-disposition is set, but only after a redirect
	if requestDidRedirect {
		fileName = filenameFromDisposition(res.Header.Get("content-disposition"))
	}

	if baseName == "." || baseName == "/" {
		// Still no value, so we'll fallback to a uuid
		if fileName == "" {
			fileName = uuid.UUID.String(uuid.New())
		}
	} else if fileName == "" {
		fileName = baseName
	}

	filePath := filepath.Join(outputPath, fileName)
	w, err := os.Create(filePath) //nolint:gosec // G304: filePath validated by caller
	if err != nil {
		return storage.StorageVolume{}, fmt.Errorf("failed to create file %s: %s", filePath, err)
	}

	defer closer.CloseWithLogOnError("file", w)

	// stream the body to the client without fully loading it into memory
	if _, err := io.Copy(w, res.Body); err != nil {
		return storage.StorageVolume{}, fmt.Errorf("failed to write to file %s: %s", filePath, err)
	}

	if err := w.Sync(); err != nil {
		return storage.StorageVolume{}, fmt.Errorf("failed to sync file %s: %w", filePath, err)
	}

	targetPath := filepath.Join(input.Target, fileName)

	log.Ctx(ctx).Debug().
		Stringer("url", u).
		Stringer("final-url", res.Request.URL).
		Str("file", filePath).
		Str("targetFile", targetPath).
		Msg("Downloaded file")

	volume := storage.StorageVolume{
		Type:   storage.StorageVolumeConnectorBind,
		Source: filePath,   // The source is the full path to the file
		Target: targetPath, // So we should alter the target to include the file name
	}

	return volume, nil
}

func filenameFromDisposition(contentDispositionHdr string) string {
	// After a redirect, when we need a filename, sometimes the server is giving
	// us a filename. We should use it.
	// We don't really care about disposition (attachment, inline) here and if
	// it is anything else then something has really gone wrong.
	fileName := ""
	if contentDispositionHdr != "" {
		_, params, err := mime.ParseMediaType(contentDispositionHdr)
		if err == nil {
			// In cases where we fail (err != nil) then we will just degrade to the
			// previous logic, but if we can find the filename, we'll set basename
			// to that.
			fileName = params["filename*"]
			if fileName == "" {
				fileName = params["filename"]
			}

			if fileName != "" {
				fileName = filepath.Base(fileName)
			}
		}
	}
	return fileName
}

func (sp *StorageProvider) CleanupStorage(
	ctx context.Context,
	_ models.InputSource,
	volume storage.StorageVolume,
) error {
	pathToCleanup := filepath.Dir(volume.Source)
	log.Ctx(ctx).Debug().Str("ResultPath", pathToCleanup).Msg("Cleaning up")
	return os.RemoveAll(pathToCleanup)
}

func (sp *StorageProvider) Upload(context.Context, string) (models.SpecConfig, error) {
	// we don't "upload" anything to a URL
	return models.SpecConfig{}, fmt.Errorf("not implemented")
}

func IsURLSupported(rawURL string) (*url.URL, error) {
	rawURL = strings.Trim(rawURL, " '\"")
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %s", err)
	}
	if (u.Scheme != "http") && (u.Scheme != "https") {
		return nil, fmt.Errorf("URLs must begin with 'http' or 'https'. The submitted one began with %s", u.Scheme)
	}

	basePath := path.Base(u.Path)

	// Need to check for both because a bare host
	// Like http://localhost/ gets converted to "." by path.Base
	if basePath == "" || u.Path == "" {
		return nil, fmt.Errorf("URL must end with a file name")
	}

	return u, nil
}

var _ storage.Storage = (*StorageProvider)(nil)

var _ retryablehttp.LeveledLogger = retryLogger{}

// This logger needs to change to fetch the logger from the context once
// https://github.com/hashicorp/go-retryablehttp/issues/182 is implemented and released.
type retryLogger struct {
}

func (r retryLogger) Error(msg string, keysAndValues ...interface{}) {
	parseKeysAndValues(log.Error(), keysAndValues...).Msg(msg)
}

func (r retryLogger) Info(msg string, keysAndValues ...interface{}) {
	parseKeysAndValues(log.Info(), keysAndValues...).Msg(msg)
}

func (r retryLogger) Debug(msg string, keysAndValues ...interface{}) {
	parseKeysAndValues(log.Debug(), keysAndValues...).Msg(msg)
}

func (r retryLogger) Warn(msg string, keysAndValues ...interface{}) {
	parseKeysAndValues(log.Warn(), keysAndValues...).Msg(msg)
}

func parseKeysAndValues(e *zerolog.Event, keysAndValues ...interface{}) *zerolog.Event {
	for i := 0; i < len(keysAndValues); i = i + 2 {
		name := keysAndValues[i].(string)
		value := keysAndValues[i+1]
		if v, ok := value.(string); ok {
			e = e.Str(name, v)
		} else if v, ok := value.(error); ok {
			e = e.AnErr(name, v)
		} else if v, ok := value.(fmt.Stringer); ok {
			e = e.Stringer(name, v)
		} else if v, ok := value.(int); ok {
			e = e.Int(name, v)
		} else {
			e = e.Interface(name, value)
		}
	}
	return e
}
