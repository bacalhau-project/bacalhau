package noop

import (
	"context"

	"github.com/bacalhau-project/bacalhau/pkg/model"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
)

type StorageHandlerIsInstalled func(ctx context.Context) (bool, error)
type StorageHandlerHasStorageLocally func(ctx context.Context, volume model.StorageSpec) (bool, error)
type StorageHandlerGetVolumeSize func(ctx context.Context, volume model.StorageSpec) (uint64, error)
type StorageHandlerPrepareStorage func(ctx context.Context, storageSpec model.StorageSpec) (storage.StorageVolume, error)
type StorageHandlerCleanupStorage func(ctx context.Context, storageSpec model.StorageSpec, volume storage.StorageVolume) error
type StorageHandlerUpload func(ctx context.Context, localPath string) (model.StorageSpec, error)
type StorageHandlerExplode func(ctx context.Context, storageSpec model.StorageSpec) ([]model.StorageSpec, error)

type StorageConfigExternalHooks struct {
	IsInstalled       StorageHandlerIsInstalled
	HasStorageLocally StorageHandlerHasStorageLocally
	GetVolumeSize     StorageHandlerGetVolumeSize
	PrepareStorage    StorageHandlerPrepareStorage
	CleanupStorage    StorageHandlerCleanupStorage
	Upload            StorageHandlerUpload
	Explode           StorageHandlerExplode
}

type StorageConfig struct {
	ExternalHooks StorageConfigExternalHooks
}

// a storage driver runs the downloads content
// from a remote ipfs server and copies it to
// to a local directory in preparation for
// a job to run - it will remove the folder/file once complete

type NoopStorage struct {
	Config StorageConfig
}

func NewNoopStorage() *NoopStorage {
	storageHandler := &NoopStorage{
		Config: StorageConfig{},
	}
	return storageHandler
}

func NewNoopStorageWithConfig(config StorageConfig) *NoopStorage {
	storageHandler := &NoopStorage{
		Config: config,
	}
	return storageHandler
}

func (s *NoopStorage) IsInstalled(ctx context.Context) (bool, error) {
	if s.Config.ExternalHooks.IsInstalled != nil {
		handler := s.Config.ExternalHooks.IsInstalled
		return handler(ctx)
	}
	return true, nil
}

func (s *NoopStorage) HasStorageLocally(ctx context.Context, volume model.StorageSpec) (bool, error) {
	if s.Config.ExternalHooks.HasStorageLocally != nil {
		handler := s.Config.ExternalHooks.HasStorageLocally
		return handler(ctx, volume)
	}
	return true, nil
}

// we wrap this in a timeout because if the CID is not present on the network this seems to hang
func (s *NoopStorage) GetVolumeSize(ctx context.Context, volume model.StorageSpec) (uint64, error) {
	if s.Config.ExternalHooks.GetVolumeSize != nil {
		handler := s.Config.ExternalHooks.GetVolumeSize
		return handler(ctx, volume)
	}
	return 0, nil
}

func (s *NoopStorage) PrepareStorage(ctx context.Context, storageSpec model.StorageSpec) (storage.StorageVolume, error) {
	if s.Config.ExternalHooks.PrepareStorage != nil {
		handler := s.Config.ExternalHooks.PrepareStorage
		return handler(ctx, storageSpec)
	}
	return storage.StorageVolume{
		Type:   storage.StorageVolumeConnectorBind,
		Source: "test",
		Target: "test",
	}, nil
}

func (s *NoopStorage) Upload(ctx context.Context, localPath string) (model.StorageSpec, error) {
	if s.Config.ExternalHooks.Upload != nil {
		handler := s.Config.ExternalHooks.Upload
		return handler(ctx, localPath)
	}
	return model.StorageSpec{
		StorageSource: model.StorageSourceIPFS,
		CID:           "test",
		Path:          "/",
	}, nil
}

func (s *NoopStorage) Explode(ctx context.Context, spec model.StorageSpec) ([]model.StorageSpec, error) {
	if s.Config.ExternalHooks.Explode != nil {
		handler := s.Config.ExternalHooks.Explode
		return handler(ctx, spec)
	}
	return []model.StorageSpec{}, nil
}

//nolint:lll // Exception to the long rule
func (s *NoopStorage) CleanupStorage(ctx context.Context, storageSpec model.StorageSpec, volume storage.StorageVolume) error {
	if s.Config.ExternalHooks.CleanupStorage != nil {
		handler := s.Config.ExternalHooks.CleanupStorage
		return handler(ctx, storageSpec, volume)
	}
	return nil
}

// Compile time interface check:
var _ storage.Storage = (*NoopStorage)(nil)
