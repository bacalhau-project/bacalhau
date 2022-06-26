package util

import (
	"context"
	"fmt"

	"github.com/filecoin-project/bacalhau/pkg/storage"
)

func GetStorageProvider(ctx context.Context, engine string, providers map[string]storage.StorageProvider) (storage.StorageProvider, error) {
	if _, ok := providers[engine]; !ok {
		return nil, fmt.Errorf("no matching storage provider found: %s", engine)
	}

	storageProvider := providers[engine]
	installed, err := storageProvider.IsInstalled(ctx)
	if err != nil {
		return nil, err
	}
	if !installed {
		return nil, fmt.Errorf("Storage provider is not installed: %s.", engine)
	}

	return storageProvider, nil
}
