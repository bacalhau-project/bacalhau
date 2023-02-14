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
	"github.com/go-resty/resty/v2"
	"github.com/google/uuid"
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

	client := resty.New()
	// Setting output directory path, If directory not exists then resty creates one
	client.SetOutputDirectory(dir)
	// Setting the number of times to try downloading the URL
	client.SetRetryCount(config.GetDownloadURLRequestRetries())
	client.SetRetryWaitTime(time.Second * 1)
	client.AddRetryAfterErrorCondition()

	storageHandler := &StorageProvider{
		HTTPClient: client,
		LocalDir:   dir,
	}

	log.Debug().Msgf("URL download driver created with output dir: %s", dir)
	return storageHandler, nil
}

func (sp *StorageProvider) IsInstalled(context.Context) (bool, error) {
	return true, nil
}

func (sp *StorageProvider) HasStorageLocally(context.Context, model.StorageSpec) (bool, error) {
	return false, nil
}

// Could do a HEAD request and check Content-Length, but in some cases that's not guaranteed to be the real end file size
func (sp *StorageProvider) GetVolumeSize(context.Context, model.StorageSpec) (uint64, error) {
	return 0, nil
}

// For the urldownload storage provider, PrepareStorage will download the file from the URL
//
//nolint:funlen,gocyclo // TODO: refactor this function
func (sp *StorageProvider) PrepareStorage(ctx context.Context, storageSpec model.StorageSpec) (storage.StorageVolume, error) {
	u, err := IsURLSupported(storageSpec.URL)
	if err != nil {
		return storage.StorageVolume{}, err
	}

	outputPath, err := os.MkdirTemp(sp.LocalDir, "*")
	if err != nil {
		return storage.StorageVolume{}, err
	}

	sp.HTTPClient.SetTimeout(config.GetDownloadURLRequestTimeout())
	sp.HTTPClient.SetOutputDirectory(outputPath)
	sp.HTTPClient.SetDoNotParseResponse(true) // We want to stream the response to disk directly

	// Trying a check for head - just trying to fail quickly if the site is clearly wrong.
	// This MAY fail with 405 (method not allowed) if the server doesn't support HEAD which is generally
	// OK because we will fail if the server is down - so it is a best effort.
	req := sp.HTTPClient.R().SetContext(ctx)
	req = req.SetContext(ctx)
	r, err := req.Head(u.String())
	log.Debug().Msgf("HEAD request to %s returned status code %d", u.String(), r.StatusCode())
	if err != nil {
		return storage.StorageVolume{}, fmt.Errorf("failed to get headers from url (%s): %s", u.String(), err)
	}

	// Checking to see about redirect here (does not 100% work because could have rejected the HEAD request)
	finalURL := r.RawResponse.Request.URL
	if finalURL != u {
		log.Debug().Msgf("URL %s redirected to %s", u.String(), finalURL.String())
	}

	// Create a new file based on the URL
	baseName := path.Base(finalURL.Path)
	var fileName string
	if baseName == "." || baseName == "/" {
		// There is no filename in the URL, so we need to a temp one
		fileName = uuid.UUID.String(uuid.New())
	} else {
		fileName = baseName
	}

	log.Trace().Msgf("Beginning get %s to %s", finalURL, outputPath)
	r, err = req.Get(finalURL.String())
	if err != nil {
		return storage.StorageVolume{},
			fmt.Errorf("failed to begin download from url %s: %s", finalURL, err)
	}

	if r.StatusCode() != http.StatusOK {
		return storage.StorageVolume{},
			fmt.Errorf("non-200 response from URL (%s): %s", storageSpec.URL, r.Status())
	}

	filePath := filepath.Join(outputPath, fileName)
	targetPath := filepath.Join(storageSpec.Path, fileName)
	w, err := os.Create(filePath)
	if err != nil {
		return storage.StorageVolume{}, fmt.Errorf("failed to create file %s: %s", filePath, err)
	}

	// stream the body to the client without fully loading it into memory
	n, err := io.Copy(w, r.RawBody())
	if err != nil {
		return storage.StorageVolume{}, fmt.Errorf("failed to write to file %s: %s", filePath, err)
	}

	if n == 0 {
		return storage.StorageVolume{}, fmt.Errorf("no bytes written to file %s", filePath)
	}

	log.Trace().Msgf("Wrote %d bytes to %s", n, filePath)

	// Closing everything
	err = w.Sync()
	if err != nil {
		return storage.StorageVolume{}, fmt.Errorf("failed to sync file %s: %s", filePath, err)
	}

	// If path.Base isn't empty, we'll see if it got redirected to a different file name
	// and if so, we'll rename it to the original file name from the URL
	// Otherwise, we'll just use the filename we created
	var finalFileName string
	if baseName != "." && baseName != "/" {
		finalFileName = filepath.Join(outputPath, path.Base(r.RawResponse.Request.URL.Path))
	} else {
		finalFileName = filePath
	}

	fileWriteName := w.Name()
	log.Debug().Msgf("Final file name based on URL: %s", finalFileName)
	log.Debug().Msgf("Final written name: %s", fileWriteName)
	if finalFileName != fileWriteName {
		log.Debug().Msgf("Downloaded file has different name than final name - renaming: %s to %s", w.Name(), finalFileName)
		err = os.Rename(w.Name(), finalFileName)
		if err != nil {
			return storage.StorageVolume{}, fmt.Errorf("failed to rename file %s to %s: %s", w.Name(), finalFileName, err)
		}

		// Need to update filePath and targetPath to accommodate the rename
		filePath = filepath.Join(outputPath, filepath.Base(finalFileName))
		targetPath = filepath.Join(storageSpec.Path, filepath.Base(finalFileName))
	}

	r.RawBody().Close()
	w.Close()

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

// we don't "upload" anything to a URL
func (sp *StorageProvider) Upload(context.Context, string) (model.StorageSpec, error) {
	return model.StorageSpec{}, fmt.Errorf("not implemented")
}

// for the url download - explode will always result in a single item
// mounted at the path specified in the spec
func (sp *StorageProvider) Explode(_ context.Context, spec model.StorageSpec) ([]model.StorageSpec, error) {
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

// Compile time interface check:
var _ storage.Storage = (*StorageProvider)(nil)
