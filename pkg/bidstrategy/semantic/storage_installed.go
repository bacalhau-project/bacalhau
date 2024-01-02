package semantic

import (
	"github.com/bacalhau-project/bacalhau/pkg/bidstrategy"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	modelsutils "github.com/bacalhau-project/bacalhau/pkg/models/utils"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
)

func NewStorageInstalledBidStrategy(storages storage.StorageProvider) bidstrategy.SemanticBidStrategy {
	return NewProviderInstalledArrayStrategy(
		storages,
		func(j *models.Job) []string {
			return modelsutils.AllInputSourcesTypes(j)
		},
	)
}
