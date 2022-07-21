package urldownload

import (
	"context"
	"fmt"
	"io/ioutil"

	"github.com/filecoin-project/bacalhau/pkg/config"
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
	client := resty.New()

	// TODO: consolidate the various config inputs into one package otherwise they are scattered across the codebase
	dir, err := ioutil.TempDir(config.GetStoragePath(), "bacalhau-url")
	if err != nil {
		return nil, err
	}
	
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
	ctx, span := newSpan(ctx, "IsInstalled")
	defer span.End()
	return true, nil
}

// TODO: @enricorotundo check if file has been downloaded already?
func (sp *StorageProvider) HasStorageLocally(ctx context.Context, volume storage.StorageSpec) (bool, error) {
	ctx, span := newSpan(ctx, "HasStorageLocally")
	defer span.End()
	return false, nil
}

// TODO: could do HEAD request but how to deal with chucked transfer/compressed responses?
func (sp *StorageProvider) GetVolumeSize(ctx context.Context, volume storage.StorageSpec) (uint64, error) {
	return 0, nil
}

// TODO: @enricorotundo add timeouts etc.
// turns StorageSpec into StorageVolume
func (sp *StorageProvider) PrepareStorage(ctx context.Context, storageSpec storage.StorageSpec) (*storage.StorageVolume, error) {
	ctx, span := newSpan(ctx, "PrepareStorage")
	defer span.End()

	outputPath, err := ioutil.TempDir(sp.LocalDir, "*")
	if err != nil {
		return nil, err
	}
	
	header, err := sp.HTTPClient.R().Head(storageSpec.URL)
	fmt.Printf("header: %s\n\n", header)

	_, err = sp.HTTPClient.R().
		SetOutput(outputPath + "/file").
		Get(storageSpec.URL)
	if err != nil {
		return nil, err
	}

	volume := &storage.StorageVolume{
		Type:   "bind",
		Source: outputPath + "/file",
		Target: storageSpec.Path,
	}

	return volume, nil
}

// TODO: @enricorotundo
// nolint:lll // Exception to the long rule
func (sp *StorageProvider) CleanupStorage(ctx context.Context, storageSpec storage.StorageSpec, volume *storage.StorageVolume) error {
	return system.RunCommand("date", []string{})
}

func newSpan(ctx context.Context, apiName string) (context.Context, trace.Span) {
	return system.Span(ctx, "storage/url/url_download", apiName)
}

// Compile time interface check:
var _ storage.StorageProvider = (*StorageProvider)(nil)
