package copy

import (
	"context"
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/lib/math"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
	"github.com/c2h5oh/datasize"
	"github.com/pkg/errors"
	"github.com/rs/zerolog/log"
	"go.uber.org/multierr"
	"golang.org/x/exp/slices"
)

type specSize struct {
	artifact *models.InputSource
	size     datasize.ByteSize
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
	specs []*models.InputSource,
	srcType, dstType string,
	maxSingle, maxTotal datasize.ByteSize,
) (modified bool, err error) {
	srcStorage, err := provider.Get(ctx, srcType)
	if err != nil {
		err = errors.Wrapf(err, "failed to get %s storage provider", srcType)
		return
	}

	specsizes := make([]specSize, 0, len(specs))
	for _, spec := range specs {
		if spec.Source.Type != srcType {
			continue
		}

		size, rerr := srcStorage.GetVolumeSize(ctx, *spec)
		if rerr != nil {
			err = errors.Wrapf(rerr, "failed to read spec %v", spec)
			return
		}
		specsizes = append(specsizes, specSize{artifact: spec, size: datasize.ByteSize(size)})
	}

	slices.SortFunc(specsizes, func(a, b specSize) bool {
		return a.size < b.size
	})

	remainingSpace := maxTotal
	for _, spec := range specsizes {
		exactFit := spec.size == remainingSpace
		remainingSpace -= math.Min(spec.size, remainingSpace)
		if (!exactFit && remainingSpace <= 0) || maxTotal == 0 || spec.size > maxSingle {
			newSpec, rerr := Copy(ctx, provider, *spec.artifact, dstType)
			if rerr != nil {
				return modified, rerr
			}

			*spec.artifact = newSpec
			log.Ctx(ctx).Debug().
				Str("Spec", fmt.Sprint(newSpec)).
				Str("OldSource", srcType).
				Msg("Replaced spec")
			modified = true
		}
	}

	return modified, err
}

// Copy transfers data described by the passed SpecConfig into the destination
// type, and returns a new SpecConfig for the data in its new location.
func Copy(
	ctx context.Context,
	provider storage.StorageProvider,
	spec models.InputSource,
	destination string,
) (models.InputSource, error) {
	srcStorage, srcErr := provider.Get(ctx, spec.Source.Type)
	dstStorage, dstErr := provider.Get(ctx, destination)
	err := multierr.Append(srcErr, dstErr)
	if err != nil {
		return models.InputSource{}, err
	}

	volume, err := srcStorage.PrepareStorage(ctx, spec)
	if err != nil {
		err = errors.Wrapf(err, "failed to prepare %s spec", spec.Source.Type)
		return models.InputSource{}, err
	}
	defer srcStorage.CleanupStorage(ctx, spec, volume) //nolint:errcheck

	var newSpec models.SpecConfig
	newSpec, err = dstStorage.Upload(ctx, volume.Source)
	if err != nil {
		err = errors.Wrapf(err, "failed to save %s spec to %s", spec.Source.Type, destination)
	}

	return models.InputSource{
		Source: &newSpec,
		Target: spec.Target,
	}, err
}
