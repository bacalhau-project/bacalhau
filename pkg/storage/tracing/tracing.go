package tracing

import (
	"context"
	"fmt"

	"github.com/rs/zerolog/log"

	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
	"github.com/bacalhau-project/bacalhau/pkg/system"
	"github.com/bacalhau-project/bacalhau/pkg/telemetry"
	"github.com/bacalhau-project/bacalhau/pkg/util/reflection"
)

type tracingStorage struct {
	delegate storage.Storage
	name     string
}

func Wrap(delegate storage.Storage) storage.Storage {
	return &tracingStorage{
		delegate: delegate,
		name:     reflection.StructName(delegate),
	}
}

func (t *tracingStorage) IsInstalled(ctx context.Context) (bool, error) {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), fmt.Sprintf("%s.IsInstalled", t.name))
	defer span.End()

	return t.delegate.IsInstalled(ctx)
}

func (t *tracingStorage) HasStorageLocally(ctx context.Context, spec models.InputSource) (bool, error) {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), fmt.Sprintf("%s.HasStorageLocally", t.name))
	defer span.End()

	return t.delegate.HasStorageLocally(ctx, spec)
}

func (t *tracingStorage) GetVolumeSize(ctx context.Context, spec models.InputSource) (uint64, error) {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), fmt.Sprintf("%s.GetVolumeSize", t.name))
	defer span.End()

	return t.delegate.GetVolumeSize(ctx, spec)
}

func (t *tracingStorage) PrepareStorage(
	ctx context.Context,
	storageDirectory string,
	spec models.InputSource) (storage.StorageVolume, error) {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), fmt.Sprintf("%s.PrepareStorage", t.name))
	defer span.End()

	stopwatch := telemetry.Timer(ctx, jobStoragePrepareDurationMilliseconds, spec.Source.MetricAttributes()...)
	defer func() {
		dur := stopwatch()
		log.Ctx(ctx).Debug().
			Dur("duration", dur).
			Object("spec", &spec).
			Str("dir", storageDirectory).
			Msg("storage prepared")
	}()

	return t.delegate.PrepareStorage(ctx, storageDirectory, spec)
}

func (t *tracingStorage) CleanupStorage(ctx context.Context, spec models.InputSource, volume storage.StorageVolume) error {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), fmt.Sprintf("%s.CleanupStorage", t.name))
	defer span.End()

	stopwatch := telemetry.Timer(ctx, jobStorageCleanupDurationMilliseconds, spec.Source.MetricAttributes()...)
	defer func() {
		dur := stopwatch()
		log.Ctx(ctx).Debug().
			Dur("duration", dur).
			Object("spec", &spec).
			Msg("storage cleanup")
	}()

	return t.delegate.CleanupStorage(ctx, spec, volume)
}

func (t *tracingStorage) Upload(ctx context.Context, path string) (models.SpecConfig, error) {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), fmt.Sprintf("%s.Upload", t.name))
	defer span.End()

	stopwatch := telemetry.Timer(ctx, jobStorageUploadDurationMilliseconds)
	defer func() {
		dur := stopwatch()
		log.Ctx(ctx).Debug().
			Dur("duration", dur).
			Str("path", path).
			Msg("storage upload")
	}()
	return t.delegate.Upload(ctx, path)
}

var _ storage.Storage = &tracingStorage{}
