package scenario

import (
	"context"
	"net/http/httptest"
	"net/url"

	"github.com/filecoin-project/bacalhau/pkg/ipfs"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/rs/zerolog/log"
	"github.com/vincent-petithory/dataurl"
)

// A SetupStorage is a function that return a model.StorageSpec representing
// some data that has been prepared for use by a job. It is the responsibility
// of the function to ensure that the data has been set up correctly.
type SetupStorage func(
	ctx context.Context,
	driverName model.StorageSourceType,
	ipfsClients ...ipfs.Client,
) ([]model.StorageSpec, error)

// StoredText will store the passed string as a file on an IPFS node, and return
// the file name and CID in the model.StorageSpec.
func StoredText(
	fileContents string,
	mountPath string,
) SetupStorage {
	return func(ctx context.Context, driverName model.StorageSourceType, clients ...ipfs.Client) ([]model.StorageSpec, error) {
		fileCid, err := ipfs.AddTextToNodes(ctx, []byte(fileContents), clients...)
		if err != nil {
			return nil, err
		}
		inputStorageSpecs := []model.StorageSpec{
			{
				StorageSource: driverName,
				CID:           fileCid,
				Path:          mountPath,
			},
		}
		log.Ctx(ctx).Debug().Msgf("Added file with cid %s", fileCid)
		return inputStorageSpecs, nil
	}
}

// StoredFile will store the file at the passed path on an IPFS node, and return
// the file name and CID in the model.StorageSpec.
func StoredFile(
	filePath string,
	mountPath string,
) SetupStorage {
	return func(ctx context.Context, driverName model.StorageSourceType, clients ...ipfs.Client) ([]model.StorageSpec, error) {
		fileCid, err := ipfs.AddFileToNodes(ctx, filePath, clients...)
		if err != nil {
			return nil, err
		}
		inputStorageSpecs := []model.StorageSpec{
			{
				StorageSource: driverName,
				CID:           fileCid,
				Path:          mountPath,
			},
		}
		return inputStorageSpecs, nil
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
	return func(_ context.Context, _ model.StorageSourceType, _ ...ipfs.Client) ([]model.StorageSpec, error) {
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
	return func(ctx context.Context, driverName model.StorageSourceType, ipfsClients ...ipfs.Client) ([]model.StorageSpec, error) {
		return store(ctx, driverName, ipfsClients[:numberOfNodes]...)
	}
}

// ManyStores runs all of the passed setups and returns the model.StorageSpecs
// associated with all of them. If any of them fail, the error from the first to
// fail will be returned.
func ManyStores(stores ...SetupStorage) SetupStorage {
	return func(ctx context.Context, driverName model.StorageSourceType, ipfsClients ...ipfs.Client) ([]model.StorageSpec, error) {
		specs := []model.StorageSpec{}
		for _, store := range stores {
			spec, err := store(ctx, driverName, ipfsClients...)
			if err != nil {
				return specs, err
			}
			specs = append(specs, spec...)
		}
		return specs, nil
	}
}
