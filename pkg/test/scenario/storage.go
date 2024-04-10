package scenario

import (
	"context"
	"net/http/httptest"
	"net/url"
	"path/filepath"

	"github.com/vincent-petithory/dataurl"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/models/migration/legacy"
	"github.com/bacalhau-project/bacalhau/pkg/storage/inline"
)

// A SetupStorage is a function that return a model.StorageSpec representing
// some data that has been prepared for use by a job. It is the responsibility
// of the function to ensure that the data has been set up correctly.
type SetupStorage func(
	ctx context.Context,
	driverName model.StorageSourceType,
) ([]model.StorageSpec, error)

// StoredText will store the passed string as a file on an IPFS node, and return
// the file name and CID in the model.StorageSpec.
func StoredText(
	fileContents string,
	mountPath string,
) SetupStorage {
	return func(ctx context.Context, driverName model.StorageSourceType) ([]model.StorageSpec, error) {
		storage := inline.NewStorage()

		config := storage.StoreBytes([]byte(fileContents))

		spec, err := legacy.ToLegacyStorageSpec(&config)
		if err != nil {
			return nil, err
		}

		spec.Path = mountPath

		inputStorageSpecs := []model.StorageSpec{spec}
		return inputStorageSpecs, nil
	}
}

// StoredFile will store the file at the passed path inline, and return
// the file name and content/URL in the model.StorageSpec.
func StoredFile(
	filePath string,
	mountPath string,
) SetupStorage {
	return func(ctx context.Context, driverName model.StorageSourceType) ([]model.StorageSpec, error) {
		abspath, err := filepath.Abs(filePath)
		if err != nil {
			return nil, err
		}

		storage := inline.NewStorage()
		specConfig, err := storage.Upload(ctx, abspath)
		if err != nil {
			return nil, err
		}

		spec, err := legacy.ToLegacyStorageSpec(&specConfig)
		if err != nil {
			return nil, err
		}

		spec.Path = mountPath

		return []model.StorageSpec{spec}, nil
	}
}

// InlineFile will store the file directly inline in the storage spec. Unlike
// the other storage set-ups, this function loads the file immediately. This
// makes it possible to store things deeper into the Spec object without the
// test system needing to know how to prepare them.
func InlineData(data []byte) model.StorageSpec {
	return model.StorageSpec{
		StorageSource: model.StorageSourceInline,
		URL:           dataurl.EncodeBytes(data),
	}
}

// URLDownload will return a model.StorageSpec referencing a file on the passed
// HTTP test server.
func URLDownload(
	server *httptest.Server,
	urlPath string,
	mountPath string,
) SetupStorage {
	return func(_ context.Context, _ model.StorageSourceType) ([]model.StorageSpec, error) {
		finalURL, err := url.JoinPath(server.URL, urlPath)
		return []model.StorageSpec{
			{
				StorageSource: model.StorageSourceURLDownload,
				URL:           finalURL,
				Path:          mountPath,
			},
		}, err
	}
}

// PartialAdd will only store data on a subset of the nodes that it is passed.
// So if there are 5 IPFS nodes configured and PartialAdd is defined with 2,
// only the first two nodes will have data loaded.
func PartialAdd(numberOfNodes int, store SetupStorage) SetupStorage {
	return func(ctx context.Context, driverName model.StorageSourceType) ([]model.StorageSpec, error) {
		return store(ctx, driverName)
	}
}

// ManyStores runs all of the passed setups and returns the model.StorageSpecs
// associated with all of them. If any of them fail, the error from the first to
// fail will be returned.
func ManyStores(stores ...SetupStorage) SetupStorage {
	return func(ctx context.Context, driverName model.StorageSourceType) ([]model.StorageSpec, error) {
		specs := []model.StorageSpec{}
		for _, store := range stores {
			spec, err := store(ctx, driverName)
			if err != nil {
				return specs, err
			}
			specs = append(specs, spec...)
		}
		return specs, nil
	}
}
