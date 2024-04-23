package compute

import (
	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	executor_util "github.com/bacalhau-project/bacalhau/pkg/executor/util"
	"github.com/bacalhau-project/bacalhau/pkg/ipfs"
	"github.com/bacalhau-project/bacalhau/pkg/lib/provider"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
	"github.com/bacalhau-project/bacalhau/pkg/storage/inline"
	ipfs_storage "github.com/bacalhau-project/bacalhau/pkg/storage/ipfs"
	localdirectory "github.com/bacalhau-project/bacalhau/pkg/storage/local_directory"
	"github.com/bacalhau-project/bacalhau/pkg/storage/repo"
	"github.com/bacalhau-project/bacalhau/pkg/storage/tracing"
	"github.com/bacalhau-project/bacalhau/pkg/storage/url/urldownload"
)

func StorageProviders(cfg types.StorageProvidersConfig, client ipfs.Client) (storage.StorageProvider, error) {
	ipfsAPICopyStorage, err := ipfs_storage.NewStorage(client)
	if err != nil {
		return nil, err
	}

	urlDownloadStorage := urldownload.NewStorage()
	if err != nil {
		return nil, err
	}

	repoCloneStorage, err := repo.NewStorage(ipfsAPICopyStorage)
	if err != nil {
		return nil, err
	}

	inlineStorage := inline.NewStorage()

	s3Storage, err := executor_util.ConfigureS3StorageProvider()
	if err != nil {
		return nil, err
	}

	localDirectoryStorage, err := localdirectory.NewStorageProvider(localdirectory.StorageProviderParams{
		AllowedPaths: localdirectory.ParseAllowPaths(cfg.AllowListedLocalPaths),
	})
	if err != nil {
		return nil, err
	}

	var useIPFSDriver storage.Storage = ipfsAPICopyStorage

	// TODO(forrest) [refactor]: use an fx decorator to add the tracing wrapper
	sp := provider.NewMappedProvider(map[string]storage.Storage{
		models.StorageSourceIPFS:           tracing.Wrap(useIPFSDriver),
		models.StorageSourceURL:            tracing.Wrap(urlDownloadStorage),
		models.StorageSourceInline:         tracing.Wrap(inlineStorage),
		models.StorageSourceRepoClone:      tracing.Wrap(repoCloneStorage),
		models.StorageSourceRepoCloneLFS:   tracing.Wrap(repoCloneStorage),
		models.StorageSourceS3:             tracing.Wrap(s3Storage),
		models.StorageSourceLocalDirectory: tracing.Wrap(localDirectoryStorage),
	})

	return provider.NewConfiguredProvider[storage.Storage](sp, cfg.Disabled), nil
}
