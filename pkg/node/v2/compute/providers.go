package compute

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/executor"
	"github.com/bacalhau-project/bacalhau/pkg/executor/docker"
	"github.com/bacalhau-project/bacalhau/pkg/executor/wasm"
	ipfs_client "github.com/bacalhau-project/bacalhau/pkg/ipfs"
	"github.com/bacalhau-project/bacalhau/pkg/lib/provider"
	"github.com/bacalhau-project/bacalhau/pkg/models"
	"github.com/bacalhau-project/bacalhau/pkg/publisher"
	"github.com/bacalhau-project/bacalhau/pkg/publisher/ipfs"
	"github.com/bacalhau-project/bacalhau/pkg/publisher/local"
	"github.com/bacalhau-project/bacalhau/pkg/publisher/s3"
	s3pub "github.com/bacalhau-project/bacalhau/pkg/publisher/s3"
	s3helper "github.com/bacalhau-project/bacalhau/pkg/s3"
	"github.com/bacalhau-project/bacalhau/pkg/storage"
	"github.com/bacalhau-project/bacalhau/pkg/storage/inline"
	localdirectory "github.com/bacalhau-project/bacalhau/pkg/storage/local_directory"
	s3strg "github.com/bacalhau-project/bacalhau/pkg/storage/s3"
	"github.com/bacalhau-project/bacalhau/pkg/storage/url/urldownload"

	executor_config "github.com/bacalhau-project/bacalhau/pkg/config/types/v2/executor"
	publisher_config "github.com/bacalhau-project/bacalhau/pkg/config/types/v2/publisher"
	storage_config "github.com/bacalhau-project/bacalhau/pkg/config/types/v2/storage"
)

func NewEngineProvider(name string, cfg executor_config.Providers) (executor.ExecutorProvider, error) {
	providers := make(map[string]executor.Executor)
	if cfg.Docker.Enabled {
		cacheCfg := cfg.Docker.ManifestCache
		// TODO we need to pass the endpoint down to the executor via cfg.Docker.Endpoint
		// currently docker is configured from system environment variables.
		dockerExecutor, err := docker.NewExecutor(name, types.DockerCacheConfig{
			Size:      cacheCfg.Size,
			Duration:  types.Duration(cacheCfg.TTL),
			Frequency: types.Duration(cacheCfg.Refresh),
		})
		if err != nil {
			return nil, fmt.Errorf("creating docker executor: %w", err)
		}
		providers[models.EngineDocker] = dockerExecutor
	}
	if cfg.WASM.Enabled {
		wasmExecutor, err := wasm.NewExecutor()
		if err != nil {
			return nil, fmt.Errorf("creating wasm executor: %w", err)
		}
		providers[models.EngineWasm] = wasmExecutor
	}

	return provider.NewMappedProvider(providers), nil
}

func NewStorageProvider(cfg storage_config.Providers) (storage.StorageProvider, error) {
	providers := make(map[string]storage.Storage)

	// TODO(forrest) [unsure]: do we intend to continue supporting inlinde data?
	// for the sake of compatibility, we'll keep this here for now.
	providers[models.StorageSourceInline] = inline.NewStorage()

	// NB(forrest): these params are the existing defaults
	maxRetries := 3
	getVoluemTimeout := time.Minute * 2

	if cfg.HTTP.Enabled {
		providers[models.StorageSourceURL] = urldownload.NewStorage(getVoluemTimeout, maxRetries)
	}
	if cfg.Local.Enabled {
		volumes := make([]localdirectory.AllowedPath, len(cfg.Local.Volumes))
		for i, v := range cfg.Local.Volumes {
			volumes[i] = localdirectory.AllowedPath{
				Path:      v.Path,
				ReadWrite: v.Write,
				// TODO this isn't currently a supported field
				// Name: v.Name
			}
		}
		localStrg, err := localdirectory.NewStorageProvider(localdirectory.StorageProviderParams{
			AllowedPaths: volumes,
		})
		if err != nil {
			return nil, fmt.Errorf("creating local storage provider: %w", err)
		}
		providers[models.StorageSourceLocalDirectory] = localStrg
	}
	if cfg.S3.Enabled {
		s3cfg, err := s3helper.DefaultAWSConfig()
		if err != nil {
			return nil, fmt.Errorf("reading S3 credentials: %w", err)
		}
		clientProvider := s3helper.NewClientProvider(s3helper.ClientProviderParams{
			AWSConfig: s3cfg,
		})
		providers[models.StorageSourceS3] = s3strg.NewStorage(getVoluemTimeout, clientProvider)
	}

	return provider.NewMappedProvider(providers), nil
}

func NewPublisherProvider(path string, cfg publisher_config.Providers) (publisher.PublisherProvider, error) {
	if path == "TODO" {
		panic("forrest needs to define a publisher path")
	}
	providers := make(map[string]publisher.Publisher)
	ctx := context.TODO()

	if cfg.S3.Enabled {
		dir, err := os.MkdirTemp(path, "bacalhau-s3-publisher")
		if err != nil {
			return nil, err
		}

		// TODO cleaning up the publisher paths can be the responsibility of the node during shutdown.
		/*
			cm.RegisterCallback(func() error {
				if err := os.RemoveAll(dir); err != nil {
					return fmt.Errorf("unable to clean up S3 publisher directory: %w", err)
				}
				return nil
			})
		*/

		// TODO the new config has many fields to configure s3, but we can't use them here
		// or will with interfere with existing behavior of fetching s3 credentials.
		awsCfg, err := s3helper.DefaultAWSConfig()
		if err != nil {
			return nil, err
		}

		providers[models.PublisherS3] = s3pub.NewPublisher(s3.PublisherParams{
			LocalDir: dir,
			ClientProvider: s3helper.NewClientProvider(s3helper.ClientProviderParams{
				AWSConfig: awsCfg,
			}),
		})
	}

	// TODO(forrest) [fubar] well this is all sorts of @#$!%#
	if cfg.LocalHTTPServer.Enabled {
		dir, err := os.MkdirTemp(path, "bacalhau-localHTTPServer-publisher")
		if err != nil {
			return nil, err
		}
		providers[models.PublisherLocal] = local.NewLocalPublisher(
			ctx,
			dir,
			cfg.LocalHTTPServer.Host,
			cfg.LocalHTTPServer.Port,
		)
	}

	if cfg.IPFS.Enabled {
		ipfsClient, err := ipfs_client.NewClient(ctx, cfg.IPFS.Endpoint)
		if err != nil {
			return nil, fmt.Errorf("creating ipfs publisher client: %w", err)
		}
		ipfsPublisher, err := ipfs.NewIPFSPublisher(ctx, *ipfsClient)
		if err != nil {
			return nil, fmt.Errorf("creatiiing ipfs publisher provider: %w", err)
		}
		providers[models.PublisherIPFS] = ipfsPublisher
	}

	return provider.NewMappedProvider(providers), nil
}

/*
func SetupAuthenticators(path string, cfg v2.Bacalhau) (authn.Provider, error) {
	if path == "TODO" {
		panic("forrest needs to define a path to the user key")
	}
	var allErr error
	privKey, allErr := loadUserIDKey(path)
	if allErr != nil {
		return nil, allErr
	}

	authns := make(map[string]authn.Authenticator, len(cfg.Server.Auth.Methods))
	for name, authnConfig := range cfg.Server.Auth.Methods {
		switch authnConfig.Type {
		case authn.MethodTypeChallenge:
			methodPolicy, err := policy.FromPathOrDefault(authnConfig.PolicyPath, challenge.AnonymousModePolicy)
			if err != nil {
				allErr = errors.Join(allErr, err)
				continue
			}

			authns[name] = challenge.NewAuthenticator(
				methodPolicy,
				challenge.NewStringMarshaller(cfg.Name),
				privKey,
				cfg.Name,
			)
		case authn.MethodTypeAsk:
			methodPolicy, err := policy.FromPath(authnConfig.PolicyPath)
			if err != nil {
				allErr = errors.Join(allErr, err)
				continue
			}

			authns[name] = ask.NewAuthenticator(
				methodPolicy,
				privKey,
				cfg.Name,
			)
		default:
			allErr = errors.Join(allErr, fmt.Errorf("unknown authentication type: %q", authnConfig.Type))
		}
	}

	return provider.NewMappedProvider(authns), allErr
}


*/
