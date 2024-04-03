package nodefx

import (
	"fmt"
	"strings"

	"github.com/bacalhau-project/bacalhau/pkg/lib/provider"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
	"github.com/bacalhau-project/bacalhau/pkg/storage/inline"
	ipfs_storage "github.com/bacalhau-project/bacalhau/pkg/storage/ipfs"
	localdirectory "github.com/bacalhau-project/bacalhau/pkg/storage/local_directory"
	s3storage "github.com/bacalhau-project/bacalhau/pkg/storage/s3"
	"github.com/bacalhau-project/bacalhau/pkg/storage/url/urldownload"
)

func StorageProviders(cfg *ComputeConfig) (storage.StorageProvider, error) {
	var (
		provided = make(map[string]storage.Storage)
		err      error
	)

	c := cfg.Providers.Storage
	for name, config := range c {
		switch strings.ToLower(name) {
		case models.StorageSourceIPFS:
			provided[name], err = IPFSStorage(config)
		case models.StorageSourceURL:
			provided[name], err = URLStorage(config)
		case models.StorageSourceInline:
			provided[name], err = InlineStorage(config)
		case models.StorageSourceS3:
			provided[name], err = S3Storage(config)
		case models.StorageSourceLocalDirectory:
			provided[name], err = LocalStorage(config)
		default:
			return nil, fmt.Errorf("unknown storage provider: %s", name)
		}
		if err != nil {
			return nil, fmt.Errorf("registering %s storeage: %w", name, err)
		}
	}
	return provider.NewMappedProvider(provided), nil
}

func IPFSStorage(cfg []byte) (*ipfs_storage.StorageProvider, error) {
	panic("TODO")
}

func URLStorage(cfg []byte) (*urldownload.StorageProvider, error) {
	panic("TODO")
}

func InlineStorage(cfg []byte) (*inline.InlineStorage, error) {
	panic("TODO")
}

func S3Storage(cfg []byte) (*s3storage.StorageProvider, error) {
	panic("TODO")
}

func LocalStorage(cfg []byte) (*localdirectory.StorageProvider, error) {
	panic("TODO")
}
