package scenario

import (
	"context"
	"net/http/httptest"
	"net/url"

	"github.com/ipfs/go-cid"

	"github.com/bacalhau-project/bacalhau/pkg/ipfs"
	"github.com/bacalhau-project/bacalhau/pkg/model/spec"
	"github.com/bacalhau-project/bacalhau/pkg/model/spec/storage/inline"
	spec_ipfs "github.com/bacalhau-project/bacalhau/pkg/model/spec/storage/ipfs"
	spec_url "github.com/bacalhau-project/bacalhau/pkg/model/spec/storage/url"

	"github.com/rs/zerolog/log"
	"github.com/vincent-petithory/dataurl"
)

// A SetupStorage is a function that return a model.StorageSpec representing
// some data that has been prepared for use by a job. It is the responsibility
// of the function to ensure that the data has been set up correctly.
type SetupStorage func(
	ctx context.Context,
	ipfsClients ...ipfs.Client,
) ([]spec.Storage, error)

// StoredText will store the passed string as a file on an IPFS node, and return
// the file name and CID in the model.StorageSpec.
func StoredText(
	fileContents string,
	mountPath string,
) SetupStorage {
	return func(ctx context.Context, clients ...ipfs.Client) ([]spec.Storage, error) {
		fileCid, err := ipfs.AddTextToNodes(ctx, []byte(fileContents), clients...)
		if err != nil {
			return nil, err
		}
		c, err := cid.Decode(fileCid)
		if err != nil {
			return nil, err
		}
		ipfsspec, err := (&spec_ipfs.IPFSStorageSpec{CID: c}).AsSpec("TODO", mountPath)
		if err != nil {
			return nil, err
		}
		log.Ctx(ctx).Debug().Msgf("Added file with cid %s", c)
		return []spec.Storage{ipfsspec}, nil
	}
}

// StoredFile will store the file at the passed path on an IPFS node, and return
// the file name and CID in the model.StorageSpec.
func StoredFile(
	filePath string,
	mountPath string,
) SetupStorage {
	return func(ctx context.Context, clients ...ipfs.Client) ([]spec.Storage, error) {
		fileCid, err := ipfs.AddFileToNodes(ctx, filePath, clients...)
		if err != nil {
			return nil, err
		}
		c, err := cid.Decode(fileCid)
		if err != nil {
			return nil, err
		}
		ipfsspec, err := (&spec_ipfs.IPFSStorageSpec{CID: c}).AsSpec("TODO", mountPath)
		if err != nil {
			return nil, err
		}
		return []spec.Storage{ipfsspec}, nil
	}
}

// InlineFile will store the file directly inline in the storage spec. Unlike
// the other storage set-ups, this function loads the file immediately. This
// makes it possible to store things deeper into the Spec object without the
// test system needing to know how to prepare them.
func InlineData(data []byte) spec.Storage {
	out, err := (&inline.InlineStorageSpec{URL: dataurl.EncodeBytes(data)}).AsSpec("TODO", "TODO")
	if err != nil {
		panic(err)
	}
	return out
}

// URLDownload will return a model.StorageSpec referencing a file on the passed
// HTTP test server.
func URLDownload(
	server *httptest.Server,
	urlPath string,
	mountPath string,
) SetupStorage {
	return func(_ context.Context, _ ...ipfs.Client) ([]spec.Storage, error) {
		finalURL, err := url.JoinPath(server.URL, urlPath)
		if err != nil {
			return nil, err
		}
		urlspec, err := (&spec_url.URLStorageSpec{URL: finalURL}).AsSpec("TODO", mountPath)
		if err != nil {
			return nil, err
		}
		return []spec.Storage{urlspec}, nil
	}
}

// PartialAdd will only store data on a subset of the nodes that it is passed.
// So if there are 5 IPFS nodes configured and PartialAdd is defined with 2,
// only the first two nodes will have data loaded.
func PartialAdd(numberOfNodes int, store SetupStorage) SetupStorage {
	return func(ctx context.Context, ipfsClients ...ipfs.Client) ([]spec.Storage, error) {
		return store(ctx, ipfsClients[:numberOfNodes]...)
	}
}

// ManyStores runs all of the passed setups and returns the model.StorageSpecs
// associated with all of them. If any of them fail, the error from the first to
// fail will be returned.
func ManyStores(stores ...SetupStorage) SetupStorage {
	return func(ctx context.Context, ipfsClients ...ipfs.Client) ([]spec.Storage, error) {
		var specs []spec.Storage
		for _, store := range stores {
			spec, err := store(ctx, ipfsClients...)
			if err != nil {
				return specs, err
			}
			specs = append(specs, spec...)
		}
		return specs, nil
	}
}
