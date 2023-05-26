// Package inline provides a storage abstraction that stores data for use
// by Bacalhau jobs within the storage spec itself, without needing any
// connection to an external storage provider.
//
// It does this (currently) by encoding the data as a RFC 2397 "data:" URL, in
// Base64 encoding. The data may be transparently compressed using Gzip
// compression if the storage system thinks this would be sensible.
//
// This helps us meet a number of use cases:
//
//  1. Providing "context" to jobs from the local filesystem as a more convenient
//     way of sharing data with jobs than having to upload to IPFS first. This is
//     useful for e.g. sharing a script to be executed by a generic job.
//  2. When we support encryption, it will be safer to transmit encrypted secrets
//     inline with the job spec itself rather than committing them to a public
//     storage space like IPFS. (They could be redacted in job listings.)
//  3. For clients running the SDK or in constrained (e.g IoT) environments, it
//     will be easier to interact with just the Bacalhau SDK than also having to
//     first persist storage and wait for this to complete. E.g. an IoT client
//     could submit some data it has collected directly to the requestor node.
//
// The storage system doesn't enforce any maximum size of the stored data. It is
// up to the rest of the system to pick a limit it thinks is suitable and
// enforce it. This is so that e.g. a requestor node can decide that an inline
// payload is too large and commit the data to IPFS instead, which would be out
// of the scope of this package.
package inline

import (
	"bytes"
	"context"
	"os"
	"path/filepath"

	"github.com/c2h5oh/datasize"
	"github.com/vincent-petithory/dataurl"
	"go.uber.org/multierr"

	"github.com/bacalhau-project/bacalhau/pkg/model/spec"
	spec_inline "github.com/bacalhau-project/bacalhau/pkg/model/spec/storage/inline"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
	"github.com/bacalhau-project/bacalhau/pkg/util/targzip"
)

// The maximum size that will be stored inline without gzip compression.
const maximumPlaintextSize datasize.ByteSize = 512 * datasize.B

// The MIME type that will be used to identify inline data that has been
// compressed. There are many different MIME types for Gzip (and in fact it's
// not regarded as a file format in of itself) but this one apparently is most
// prevalent (see https://superuser.com/q/901962)
const gzipMimeType string = "application/gzip"

type InlineStorage struct{}

func NewStorage() *InlineStorage {
	return &InlineStorage{}
}

// As PrepareStorage writes the data to the local filesystem, CleanupStorage
// just needs to remove that temporary directory.
func (i *InlineStorage) CleanupStorage(_ context.Context, _ spec.Storage, vol storage.StorageVolume) error {
	return os.RemoveAll(vol.Source)
}

// For an inline storage, we define the volume size as uncompressed data size,
// as this is how much resource using the storage will take up.
func (i *InlineStorage) GetVolumeSize(_ context.Context, strgSpec spec.Storage) (uint64, error) {
	inlinespec, err := spec_inline.Decode(strgSpec)
	if err != nil {
		return 0, err
	}
	data, err := dataurl.DecodeString(inlinespec.URL)
	if err != nil {
		return 0, err
	}

	if data.ContentType() == gzipMimeType {
		size, derr := targzip.UncompressedSize(bytes.NewReader(data.Data))
		return size.Bytes(), derr
	} else {
		return uint64(len(data.Data)), nil
	}
}

// The storage is always local because it is contained with the StorageSpec.
func (*InlineStorage) HasStorageLocally(context.Context, spec.Storage) (bool, error) {
	return true, nil
}

// The storage is always installed because it has no external dependencies.
func (*InlineStorage) IsInstalled(context.Context) (bool, error) {
	return true, nil
}

// PrepareStorage extracts the data from the "data:" URL and writes it to a
// temporary directory. If the data was a compressed tarball, it decompresses it
// into a directory structure.
func (i *InlineStorage) PrepareStorage(_ context.Context, strgSpec spec.Storage) (storage.StorageVolume, error) {
	inlinespec, err := spec_inline.Decode(strgSpec)
	if err != nil {
		return storage.StorageVolume{}, err
	}

	tempdir, err := os.MkdirTemp(os.TempDir(), "inline-storage")
	if err != nil {
		return storage.StorageVolume{}, err
	}

	data, err := dataurl.DecodeString(inlinespec.URL)
	if err != nil {
		return storage.StorageVolume{}, err
	}

	reader := bytes.NewReader(data.Data)
	if data.ContentType() == gzipMimeType {
		err = os.Remove(tempdir)
		if err != nil {
			return storage.StorageVolume{}, err
		}

		err = targzip.Decompress(reader, tempdir)
		return storage.StorageVolume{
			Type:   storage.StorageVolumeConnectorBind,
			Source: tempdir,
			Target: strgSpec.Mount,
		}, err
	} else {
		tempfile, err := os.CreateTemp(tempdir, "file")
		if err != nil {
			return storage.StorageVolume{}, err
		}

		_, werr := tempfile.Write(data.Data)
		cerr := tempfile.Close()
		return storage.StorageVolume{
			Type:   storage.StorageVolumeConnectorBind,
			Source: tempfile.Name(),
			Target: strgSpec.Mount,
		}, multierr.Combine(werr, cerr)
	}
}

// Upload stores the data into the returned StorageSpec. If the path points to a
// directory, the directory will be made into a tarball. The data might be
// compressed and will always be base64-encoded using a URL-safe method.
func (*InlineStorage) Upload(ctx context.Context, path string) (spec.Storage, error) {
	info, err := os.Stat(path)
	if err != nil {
		return spec.Storage{}, err
	}

	var url string
	if info.IsDir() || info.Size() > int64(maximumPlaintextSize.Bytes()) {
		cwd, err := os.Getwd()
		if err != nil {
			return spec.Storage{}, err
		}
		if err := os.Chdir(filepath.Dir(path)); err != nil {
			return spec.Storage{}, err
		}
		var buf bytes.Buffer
		if err := targzip.Compress(ctx, filepath.Base(path), &buf); err != nil {
			return spec.Storage{}, err
		}
		url = dataurl.New(buf.Bytes(), gzipMimeType).String()
		if err := os.Chdir(cwd); err != nil {
			return spec.Storage{}, err
		}
	} else {
		data, err := os.ReadFile(path)
		if err != nil {
			return spec.Storage{}, err
		}
		url = dataurl.EncodeBytes(data)
	}

	// TODO what is the name and mount of this storage supposed to be?
	return (&spec_inline.InlineStorageSpec{URL: url}).AsSpec("TODO", "TODO")
}

var _ storage.Storage = (*InlineStorage)(nil)
