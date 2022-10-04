package urldownload

import (
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"

	"github.com/filecoin-project/bacalhau/pkg/config"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/go-resty/resty/v2"
	"github.com/rs/zerolog/log"
)

// a storage driver runs the downloads content
// from a public URL source and copies it to
// a local directory in preparation for
// a job to run - it will remove the folder/file once complete

type StorageProvider struct {
	LocalDir   string
	HTTPClient *resty.Client
}

func NewStorage(cm *system.CleanupManager) (*StorageProvider, error) {
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
	return true, nil
}

func (sp *StorageProvider) HasStorageLocally(ctx context.Context, volume model.StorageSpec) (bool, error) {
	return false, nil
}

// Could do a HEAD request and check Content-Length, but in some cases that's not guaranteed to be the real end file size
func (sp *StorageProvider) GetVolumeSize(ctx context.Context, volume model.StorageSpec) (uint64, error) {
	return 0, nil
}

// For the urldownload storage provider, PrepareStorage will download the file from the URL
func (sp *StorageProvider) PrepareStorage(ctx context.Context, storageSpec model.StorageSpec) (storage.StorageVolume, error) {
	_, span := system.GetTracer().Start(ctx, "pkg/storage/url/urldownload.PrepareStorage")
	defer span.End()

	u, err := IsURLSupported(storageSpec.URL)
	if err != nil {
		return storage.StorageVolume{}, err
	}

	outputPath, err := ioutil.TempDir(sp.LocalDir, "*")
	if err != nil {
		return storage.StorageVolume{}, err
	}

	sp.HTTPClient.SetTimeout(config.GetDownloadURLRequestTimeout())
	sp.HTTPClient.SetOutputDirectory(outputPath)
	sp.HTTPClient.SetDoNotParseResponse(true) // We want to stream the response to disk directly

	req := sp.HTTPClient.R().SetContext(ctx)
	req = req.SetContext(ctx)
	r, err := req.Head(u.String())
	log.Debug().Msgf("HEAD request to %s returned status code %d", u.String(), r.StatusCode())
	if err != nil {
		return storage.StorageVolume{}, fmt.Errorf("failed to get headers from url (%s): %s", u.String(), err)
	}

	log.Trace().Msgf("Beginning get %s to %s", u.String(), outputPath)
	r, err = req.Get(u.String())
	if err != nil {
		return storage.StorageVolume{}, fmt.Errorf("failed to begin download from url %s: %s", u.String(), err)
	}

	if r.StatusCode() != http.StatusOK {
		return storage.StorageVolume{}, fmt.Errorf("non-200 response from URL (%s): %s", storageSpec.URL, r.Status())
	}

	// Create a new file based on the URL
	fileName := filepath.Base(path.Base(u.Path))
	filePath := filepath.Join(outputPath, fileName)
	w, err := os.Create(filePath)
	if err != nil {
		return storage.StorageVolume{}, fmt.Errorf("failed to create file %s: %s", filePath, err)
	}

	// stream the body to the client without fully loading it into memory
	n, err := io.Copy(w, r.RawBody())
	if err != nil {
		return storage.StorageVolume{}, fmt.Errorf("failed to write to file %s: %s", filePath, err)
	}

	log.Trace().Msgf("Wrote %d bytes to %s", n, filePath)

	// Closing everything
	err = w.Sync()
	if err != nil {
		return storage.StorageVolume{}, fmt.Errorf("failed to sync file %s: %s", filePath, err)
	}
	r.RawBody().Close()
	w.Close()

	volume := storage.StorageVolume{
		Type:   storage.StorageVolumeConnectorBind,
		Source: outputPath,
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
	_, span := system.GetTracer().Start(ctx, "pkg/storage/url/urldownload.CleanupStorage")
	defer span.End()

	pathToCleanup := filepath.Dir(volume.Source)
	log.Debug().Msgf("Cleaning up: %s", pathToCleanup)

	_, err := system.UnsafeForUserCodeRunCommand("rm", []string{
		"-rf", pathToCleanup,
	})

	if err != nil {
		return err
	}
	return nil
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
			Name:          spec.Name,
			StorageSource: model.StorageSourceURLDownload,
			Path:          spec.Path,
			URL:           spec.URL,
		},
	}, nil
}

func IsURLSupported(rawURL string) (*url.URL, error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("invalid URL: %s", err)
	}
	if (u.Scheme != "http") && (u.Scheme != "https") {
		return nil, fmt.Errorf("URLs must begin with 'http' or 'https'. The submitted one began with %s", u.Scheme)
	}

	return u, nil
}

// Compile time interface check:
var _ storage.Storage = (*StorageProvider)(nil)
