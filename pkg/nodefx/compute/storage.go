package compute

import (
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	executor_util "github.com/bacalhau-project/bacalhau/pkg/executor/util"
	"github.com/bacalhau-project/bacalhau/pkg/ipfs"
	"github.com/bacalhau-project/bacalhau/pkg/lib/provider"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
)

func StorageProviders(cfg types.StorageProvidersConfig, client ipfs.Client) (storage.StorageProvider, error) {
	pr, err := executor_util.NewStandardStorageProvider(
		executor_util.StandardStorageProviderOptions{
			API:                   client,
			AllowListedLocalPaths: cfg.AllowListedLocalPaths,
		},
	)
	if err != nil {
		return nil, err
	}
	// NB(forrest): yet another provider providing providers
	return provider.NewConfiguredProvider(pr, cfg.Disabled), err

}

// NB(forrest) a very rough idea on how we can support pluggable providers
/*
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
*/
