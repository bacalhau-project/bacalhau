package copy

import (
	"context"
	"fmt"

	"github.com/c2h5oh/datasize"
	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"go.uber.org/multierr"
	"golang.org/x/exp/slices"
)

type specSize struct {
	spec *model.StorageSpec
	size datasize.ByteSize
}

// CopyOversize transfers StorageSpecs from one StorageSourceType to another in
// order to fit the specs into the passed size limits.
//
// A spec will be transferred if it is over the passed maxSingle size. It may be
// transferred if all the specs are over the passed maxTotal size, depending on
// how big the other specs are (bigger specs are transferred first).
//
// The specs will be updated in place to contain the location of the new data.
// If any specs are not of the passed srcType, they are ignored.
//
// Passing 0 as either limit will cause all specs to be transferred.
func CopyOversize(
	ctx context.Context,
	provider storage.StorageProvider,
	specs []*model.StorageSpec,
	srcType, dstType model.StorageSourceType,
	maxSingle, maxTotal datasize.ByteSize,
) (modified bool, err error) {
	srcStorage, err := provider.Get(ctx, srcType)
	if err != nil {
		err = errors.Wrapf(err, "failed to get %s storage provider", srcType)
		return
	}

	specsizes := make([]specSize, 0, len(specs))
	for _, spec := range specs {
		if spec.StorageSource != srcType {
			continue
		}

		size, rerr := srcStorage.GetVolumeSize(ctx, *spec)
		if rerr != nil {
			err = errors.Wrapf(rerr, "failed to read spec %v", spec)
			return
		}
		specsizes = append(specsizes, specSize{spec: spec, size: datasize.ByteSize(size)})
	}

	slices.SortFunc(specsizes, func(a, b specSize) bool {
		return a.size < b.size
	})

	remainingSpace := maxTotal
	for _, spec := range specsizes {
		exactFit := spec.size == remainingSpace
		remainingSpace -= system.Min(spec.size, remainingSpace)
		if (!exactFit && remainingSpace <= 0) || maxTotal == 0 || spec.size > maxSingle {
			newSpec, rerr := Copy(ctx, provider, *spec.spec, dstType)
			if rerr != nil {
				return modified, rerr
			}

			*spec.spec = newSpec
			log.Ctx(ctx).Debug().
				Str("Spec", fmt.Sprint(newSpec)).
				Stringer("OldSource", srcType).
				Msg("Replaced spec")
			modified = true
		}
	}

	return modified, err
}

// Copy transfers data described by the passed StorageSpec into the destination
// type, and returns a new StorageSpec for the data in its new location.
func Copy(
	ctx context.Context,
	provider storage.StorageProvider,
	spec model.StorageSpec,
	destination model.StorageSourceType,
) (model.StorageSpec, error) {
	srcStorage, srcErr := provider.Get(ctx, spec.StorageSource)
	dstStorage, dstErr := provider.Get(ctx, destination)
	err := multierr.Append(srcErr, dstErr)
	if err != nil {
		return model.StorageSpec{}, err
	}

	volume, err := srcStorage.PrepareStorage(ctx, spec)
	if err != nil {
		err = errors.Wrapf(err, "failed to prepare %s spec", spec.StorageSource)
		return model.StorageSpec{}, err
	}
	defer srcStorage.CleanupStorage(ctx, spec, volume) //nolint:errcheck

	var newSpec model.StorageSpec
	newSpec, err = dstStorage.Upload(ctx, volume.Source)
	if err != nil {
		err = errors.Wrapf(err, "failed to save %s spec to %s", spec.StorageSource, destination)
	}
	return newSpec, err
}
