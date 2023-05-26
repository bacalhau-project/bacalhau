package tracing

import (
	"context"
	"fmt"

	"github.com/bacalhau-project/bacalhau/pkg/model/spec"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
	"github.com/bacalhau-project/bacalhau/pkg/system"
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

func (t *tracingStorage) HasStorageLocally(ctx context.Context, spec spec.Storage) (bool, error) {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), fmt.Sprintf("%s.HasStorageLocally", t.name))
	defer span.End()

	return t.delegate.HasStorageLocally(ctx, spec)
}

func (t *tracingStorage) GetVolumeSize(ctx context.Context, spec spec.Storage) (uint64, error) {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), fmt.Sprintf("%s.GetVolumeSize", t.name))
	defer span.End()

	return t.delegate.GetVolumeSize(ctx, spec)
}

func (t *tracingStorage) PrepareStorage(ctx context.Context, spec spec.Storage) (storage.StorageVolume, error) {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), fmt.Sprintf("%s.PrepareStorage", t.name))
	defer span.End()

	return t.delegate.PrepareStorage(ctx, spec)
}

func (t *tracingStorage) CleanupStorage(ctx context.Context, spec spec.Storage, volume storage.StorageVolume) error {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), fmt.Sprintf("%s.CleanupStorage", t.name))
	defer span.End()

	return t.delegate.CleanupStorage(ctx, spec, volume)
}

func (t *tracingStorage) Upload(ctx context.Context, s string) (spec.Storage, error) {
	ctx, span := system.NewSpan(ctx, system.GetTracer(), fmt.Sprintf("%s.Upload", t.name))
	defer span.End()

	return t.delegate.Upload(ctx, s)
}

var _ storage.Storage = &tracingStorage{}
