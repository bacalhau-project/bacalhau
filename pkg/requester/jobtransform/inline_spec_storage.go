package jobtransform

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	modelsutils "github.com/bacalhau-project/bacalhau/pkg/models/utils"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
	"github.com/bacalhau-project/bacalhau/pkg/storage/copy"
	"github.com/c2h5oh/datasize"
	"github.com/rs/zerolog/log"
)

// The maximum size that an individual inline storage spec and all inline
// storage specs (respectively) can take up before being pinned to IPFS storage.
const (
	maximumIndividualSpec datasize.ByteSize = 5 * datasize.KB
	maximumTotalSpec      datasize.ByteSize = 5 * datasize.KB
)

// NewInlineStoragePinner returns a job transformer that limits the inline space
// taken up by inline data. It will scan a job for StorageSpec objects that
// store their data inline and move any that are too large into IPFS storage. It
// also limits the total size taken up by inline specs and if this value is
// exceeded it will move the largest specs into IPFS.
func NewInlineStoragePinner(provider storage.StorageProvider) PostTransformer {
	return func(ctx context.Context, j *models.Job) (modified bool, err error) {
		hasInline := provider.Has(ctx, model.StorageSourceInline.String())
		hasIPFS := provider.Has(ctx, model.StorageSourceIPFS.String())
		if !hasInline || !hasIPFS {
			log.Ctx(ctx).Warn().Msg("Skipping inline data transform because storage not installed")
			return false, nil
		}

		return copy.CopyOversize(
			ctx,
			provider,
			modelsutils.AllInputSources(j),
			model.StorageSourceInline.String(),
			model.StorageSourceIPFS.String(),
			maximumIndividualSpec,
			maximumTotalSpec,
		)
	}
}
