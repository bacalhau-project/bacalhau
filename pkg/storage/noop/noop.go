package noop

import (
	"context"

	"github.com/filecoin-project/bacalhau/pkg/storage"
	"github.com/filecoin-project/bacalhau/pkg/system"
)

type StroageHandlerIsInstalled func(ctx context.Context) (bool, error)
type StroageHandlerHasStorageLocally func(ctx context.Context, volume storage.StorageSpec) (bool, error)
type StroageHandlerGetVolumeSize func(ctx context.Context, volume storage.StorageSpec) (uint64, error)
type StroageHandlerPrepareStorage func(ctx context.Context, storageSpec storage.StorageSpec) (storage.StorageVolume, error)
type StroageHandlerCleanupStorage func(ctx context.Context, storageSpec storage.StorageSpec, volume storage.StorageVolume) error
type StroageHandlerUpload func(ctx context.Context, localPath string) (storage.StorageSpec, error)
type StroageHandlerExplode func(ctx context.Context, storageSpec storage.StorageSpec) ([]storage.StorageSpec, error)

type StorageConfigExternalHooks struct {
	IsInstalled       StroageHandlerIsInstalled
	HasStorageLocally StroageHandlerHasStorageLocally
	GetVolumeSize     StroageHandlerGetVolumeSize
	PrepareStorage    StroageHandlerPrepareStorage
	CleanupStorage    StroageHandlerCleanupStorage
	Upload            StroageHandlerUpload
	Explode           StroageHandlerExplode
}

type StorageConfig struct {
	ExternalHooks StorageConfigExternalHooks
}

// a storage driver runs the downloads content
// from a remote ipfs server and copies it to
// to a local directory in preparation for
// a job to run - it will remove the folder/file once complete

type StorageProvider struct {
	Config StorageConfig
}

func NewStorageProvider(cm *system.CleanupManager) (*StorageProvider, error) {
	storageHandler := &StorageProvider{}
	return storageHandler, nil
}

func NewStorageProviderWithConfig(cm *system.CleanupManager, config StorageConfig) (*StorageProvider, error) {
	storageHandler := &StorageProvider{
		Config: config,
	}
	return storageHandler, nil
}

func (s *StorageProvider) IsInstalled(ctx context.Context) (bool, error) {
	if s.Config.ExternalHooks.IsInstalled != nil {
		handler := s.Config.ExternalHooks.IsInstalled
		return handler(ctx)
	}
	return true, nil
}

func (s *StorageProvider) HasStorageLocally(ctx context.Context, volume storage.StorageSpec) (bool, error) {
	if s.Config.ExternalHooks.HasStorageLocally != nil {
		handler := s.Config.ExternalHooks.HasStorageLocally
		return handler(ctx, volume)
	}
	return true, nil
}

// we wrap this in a timeout because if the CID is not present on the network this seems to hang
func (s *StorageProvider) GetVolumeSize(ctx context.Context, volume storage.StorageSpec) (uint64, error) {
	if s.Config.ExternalHooks.GetVolumeSize != nil {
		handler := s.Config.ExternalHooks.GetVolumeSize
		return handler(ctx, volume)
	}
	return 0, nil
}

func (s *StorageProvider) PrepareStorage(ctx context.Context, storageSpec storage.StorageSpec) (storage.StorageVolume, error) {
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

func (s *StorageProvider) Upload(ctx context.Context, localPath string) (storage.StorageSpec, error) {
	if s.Config.ExternalHooks.Upload != nil {
		handler := s.Config.ExternalHooks.Upload
		return handler(ctx, localPath)
	}
	return storage.StorageSpec{
		Engine: storage.StorageSourceIPFS,
		Cid:    "test",
		Path:   "/",
	}, nil
}

func (s *StorageProvider) Explode(ctx context.Context, spec storage.StorageSpec) ([]storage.StorageSpec, error) {
	if s.Config.ExternalHooks.Explode != nil {
		handler := s.Config.ExternalHooks.Explode
		return handler(ctx, spec)
	}
	return []storage.StorageSpec{}, nil
}

//nolint:lll // Exception to the long rule
func (s *StorageProvider) CleanupStorage(ctx context.Context, storageSpec storage.StorageSpec, volume storage.StorageVolume) error {
	if s.Config.ExternalHooks.CleanupStorage != nil {
		handler := s.Config.ExternalHooks.CleanupStorage
		return handler(ctx, storageSpec, volume)
	}
	return nil
}

// Compile time interface check:
var _ storage.StorageProvider = (*StorageProvider)(nil)
