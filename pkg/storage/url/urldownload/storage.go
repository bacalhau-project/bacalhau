package urldownload

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/filecoin-project/bacalhau/pkg/config"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/util/closer"
	"github.com/google/uuid"
	"github.com/hashicorp/go-retryablehttp"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	"go.opentelemetry.io/otel/trace"
)

// a storage driver runs the downloads content
// from a public URL source and copies it to
// a local directory in preparation for
// a job to run - it will remove the folder/file once complete

type StorageProvider struct {
	localDir string
	client   *retryablehttp.Client
}

func NewStorage(cm *system.CleanupManager) (*StorageProvider, error) {
	// TODO: consolidate the various config inputs into one package otherwise they are scattered across the codebase
	dir, err := os.MkdirTemp(config.GetStoragePath(), "bacalhau-url")
	if err != nil {
		return nil, err
	}

	cm.RegisterCallback(func() error {
		if err := os.RemoveAll(dir); err != nil {
			return fmt.Errorf("unable to remove storage folder: %w", err)
		}
		return nil
	})

	log.Debug().Str("dir", dir).Msg("URL download driver created with output dir")

	return newStorage(dir), nil
}

func newStorage(dir string) *StorageProvider {
	client := retryablehttp.NewClient()
	client.HTTPClient = &http.Client{
		Timeout: config.GetDownloadURLRequestTimeout(),
		Transport: otelhttp.NewTransport(nil, otelhttp.WithSpanNameFormatter(func(operation string, r *http.Request) string {
			return fmt.Sprintf("%s %s", r.Method, r.URL.Path)
		}), otelhttp.WithSpanOptions(trace.WithAttributes(semconv.PeerService("url-download")))),
	}
	client.RetryMax = config.GetDownloadURLRequestRetries()
	client.RetryWaitMax = time.Second * 1
	client.Logger = retryLogger{}
	client.CheckRetry = func(ctx context.Context, resp *http.Response, err error) (bool, error) {
		if err := ctx.Err(); err != nil { //nolint:govet
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
		localDir: dir,
		client:   client,
	}
}

func (sp *StorageProvider) IsInstalled(context.Context) (bool, error) {
	return true, nil
}

func (sp *StorageProvider) HasStorageLocally(context.Context, model.StorageSpec) (bool, error) {
	return false, nil
}

func (sp *StorageProvider) GetVolumeSize(context.Context, model.StorageSpec) (uint64, error) {
	// Could do a HEAD request and check Content-Length, but in some cases that's not guaranteed to be the real end file size
	return 0, nil
}

// PrepareStorage will download the file from the URL
func (sp *StorageProvider) PrepareStorage(ctx context.Context, storageSpec model.StorageSpec) (storage.StorageVolume, error) {
	u, err := IsURLSupported(storageSpec.URL)
	if err != nil {
		return storage.StorageVolume{}, err
	}

	outputPath, err := os.MkdirTemp(sp.localDir, "*")
	if err != nil {
		return storage.StorageVolume{}, err
	}

	req, err := retryablehttp.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return storage.StorageVolume{}, err
	}
	res, err := sp.client.Do(req) //nolint:bodyclose // this is being closed - golangci-lint is wrong again
	if err != nil {
		return storage.StorageVolume{}, fmt.Errorf("failed to begin download from url %s: %w", u, err)
	}
	defer closer.DrainAndCloseWithLogOnError(ctx, "response", res.Body)

	if res.StatusCode < http.StatusOK || res.StatusCode >= http.StatusMultipleChoices {
		return storage.StorageVolume{}, fmt.Errorf("non-200 response from URL (%s): %s", storageSpec.URL, res.Status)
	}

	baseName := path.Base(res.Request.URL.Path)
	var fileName string
	if baseName == "." || baseName == "/" {
		// There is no filename in the URL, so we need to a temp one
		fileName = uuid.UUID.String(uuid.New())
	} else {
		fileName = baseName
	}

	filePath := filepath.Join(outputPath, fileName)
	w, err := os.Create(filePath)
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

	targetPath := filepath.Join(storageSpec.Path, fileName)

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

func (sp *StorageProvider) CleanupStorage(
	ctx context.Context,
	_ model.StorageSpec,
	volume storage.StorageVolume,
) error {
	pathToCleanup := filepath.Dir(volume.Source)
	log.Ctx(ctx).Debug().Str("Path", pathToCleanup).Msg("Cleaning up")
	return os.RemoveAll(pathToCleanup)
}

func (sp *StorageProvider) Upload(context.Context, string) (model.StorageSpec, error) {
	// we don't "upload" anything to a URL
	return model.StorageSpec{}, fmt.Errorf("not implemented")
}

func (sp *StorageProvider) Explode(_ context.Context, spec model.StorageSpec) ([]model.StorageSpec, error) {
	// for the url download - explode will always result in a single item
	// mounted at the path specified in the spec
	return []model.StorageSpec{
		{
			Name:          spec.Name,
			StorageSource: model.StorageSourceURLDownload,
			Path:          spec.Path,
			URL:           spec.URL,
		},
	}, nil
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
