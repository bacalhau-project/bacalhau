package jobtransform

import (
	"context"

	"github.com/c2h5oh/datasize"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/storage/copy"
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
func NewInlineStoragePinner(provider storage.StorageProvider) Transformer {
	return func(ctx context.Context, j *model.Job) (modified bool, err error) {
		return copy.CopyOversize(
			ctx,
			provider,
			j.Spec.AllStorageSpecs(),
			model.StorageSourceInline,
			model.StorageSourceIPFS,
			maximumIndividualSpec,
			maximumTotalSpec,
		)
	}
}
