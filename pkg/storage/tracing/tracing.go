package tracing

import (
	"context"
	"fmt"

	"github.com/filecoin-project/bacalhau/pkg/model"
	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/system"
	"github.com/filecoin-project/bacalhau/pkg/util/reflection"
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

func (t *tracingStorage) HasStorageLocally(ctx context.Context, spec model.StorageSpec) (bool, error) {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), fmt.Sprintf("%s.HasStorageLocally", t.name))
	defer span.End()

	return t.delegate.HasStorageLocally(ctx, spec)
}

func (t *tracingStorage) GetVolumeSize(ctx context.Context, spec model.StorageSpec) (uint64, error) {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), fmt.Sprintf("%s.GetVolumeSize", t.name))
	defer span.End()

	return t.delegate.GetVolumeSize(ctx, spec)
}

func (t *tracingStorage) PrepareStorage(ctx context.Context, spec model.StorageSpec) (storage.StorageVolume, error) {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), fmt.Sprintf("%s.PrepareStorage", t.name))
	defer span.End()

	return t.delegate.PrepareStorage(ctx, spec)
}

func (t *tracingStorage) CleanupStorage(ctx context.Context, spec model.StorageSpec, volume storage.StorageVolume) error {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), fmt.Sprintf("%s.CleanupStorage", t.name))
	defer span.End()

	return t.delegate.CleanupStorage(ctx, spec, volume)
}

func (t *tracingStorage) Upload(ctx context.Context, s string) (model.StorageSpec, error) {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), fmt.Sprintf("%s.Upload", t.name))
	defer span.End()

	return t.delegate.Upload(ctx, s)
}

func (t *tracingStorage) Explode(ctx context.Context, spec model.StorageSpec) ([]model.StorageSpec, error) {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), fmt.Sprintf("%s.Explode", t.name))
	defer span.End()

	return t.delegate.Explode(ctx, spec)
}

var _ storage.Storage = &tracingStorage{}
