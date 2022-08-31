package urldownload

import (
	"context"
	"fmt"
	"io/ioutil"
	"net/url"
	"path/filepath"

	"github.com/filecoin-project/bacalhau/pkg/config"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/go-resty/resty/v2"
	"github.com/rs/zerolog/log"
	"go.opentelemetry.io/otel/trace"
)

// a storage driver runs the downloads content
// from a public URL source and copies it to
// a local directory in preparation for
// a job to run - it will remove the folder/file once complete

type StorageProvider struct {
	LocalDir   string
	HTTPClient *resty.Client
}

func NewStorageProvider(cm *system.CleanupManager) (*StorageProvider, error) {
	// TODO: consolidate the various config inputs into one package otherwise they are scattered across the codebase
	dir, err := ioutil.TempDir(config.GetStoragePath(), "bacalhau-url")
	if err != nil {
		return nil, err
	}

	client := resty.New()
	// Setting output directory path, If directory not exists then resty creates one
	client.SetOutputDirectory(dir)

	storageHandler := &StorageProvider{
		HTTPClient: client,
		LocalDir:   dir,
	}

	log.Debug().Msgf("URL download driver created with output dir: %s", dir)
	return storageHandler, nil
}

func (sp *StorageProvider) IsInstalled(ctx context.Context) (bool, error) {
	_, span := newSpan(ctx, "IsInstalled")
	defer span.End()
	return true, nil
}

func (sp *StorageProvider) HasStorageLocally(ctx context.Context, volume model.StorageSpec) (bool, error) {
	_, span := newSpan(ctx, "HasStorageLocally")
	defer span.End()
	return false, nil
}

// Could do a HEAD request and check Content-Length, but in some cases that's not guaranteed to be the real end file size
func (sp *StorageProvider) GetVolumeSize(ctx context.Context, volume model.StorageSpec) (uint64, error) {
	return 0, nil
}

func (sp *StorageProvider) PrepareStorage(ctx context.Context, storageSpec model.StorageSpec) (storage.StorageVolume, error) {
	_, span := newSpan(ctx, "PrepareStorage")
	defer span.End()

	_, err := IsURLSupported(storageSpec.URL)
	if err != nil {
		return storage.StorageVolume{}, err
	}

	outputPath, err := ioutil.TempDir(sp.LocalDir, "*")
	if err != nil {
		return storage.StorageVolume{}, err
	}

	sp.HTTPClient.SetTimeout(config.GetDownloadURLRequestTimeout())
	_, err = sp.HTTPClient.R().
		SetOutput(outputPath + "/file").
		Get(storageSpec.URL)
	if err != nil {
		return storage.StorageVolume{}, err
	}

	volume := storage.StorageVolume{
		Type:   storage.StorageVolumeConnectorBind,
		Source: outputPath + "/file",
		Target: storageSpec.Path,
	}

	return volume, nil
}

// func (sp *StorageProvider) CleanupStorage(ctx context.Context, storageSpec model.StorageSpec, volume storage.StorageVolume) error {
func (sp *StorageProvider) CleanupStorage(
	ctx context.Context,
	storageSpec model.StorageSpec,
	volume storage.StorageVolume,
) error {
	pathToCleanup := filepath.Dir(volume.Source)
	log.Debug().Msgf("Cleaning up: %s", pathToCleanup)

	return system.RunCommand("rm", []string{
		"-rf", pathToCleanup,
	})
}

// we don't "upload" anything to a URL
func (sp *StorageProvider) Upload(ctx context.Context, localPath string) (model.StorageSpec, error) {
	return model.StorageSpec{}, fmt.Errorf("not implemented")
}

// for the url download - explode will always result in a single item
// mounted at the path specified in the spec
func (sp *StorageProvider) Explode(ctx context.Context, spec model.StorageSpec) ([]model.StorageSpec, error) {
	return []model.StorageSpec{
		{
			Name:   spec.Name,
			Engine: model.StorageSourceURLDownload,
			Path:   spec.Path,
			URL:    spec.URL,
		},
	}, nil
}

func IsURLSupported(rawURL string) (bool, error) {
	// The string url is assumed NOT to have a #fragment suffix
	// thus the valid form is: [scheme:][//[userinfo@]host][/]path[?query]
	parsedURL, err := url.ParseRequestURI(rawURL)
	if err != nil {
		return false, err
	}
	if (parsedURL.Scheme == "http") || (parsedURL.Scheme == "https") {
		return true, nil
	}
	return false, fmt.Errorf("protocol scheme in URL not supported: %s", rawURL)
}

func newSpan(ctx context.Context, apiName string) (context.Context, trace.Span) {
	return system.Span(ctx, "storage/url/url_download", apiName)
}

// Compile time interface check:
var _ storage.StorageProvider = (*StorageProvider)(nil)
