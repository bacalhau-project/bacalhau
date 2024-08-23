package configv2

import (
	"fmt"
	"slices"
	"strings"

	"github.com/mitchellh/mapstructure"

	v1types "github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/configv2/types"
)

func MigrateV1(in v1types.BacalhauConfig) (types.Bacalhau, error) {
	publisherConfig, err := migratePublishers(in.Node)
	if err != nil {
		return types.Bacalhau{}, fmt.Errorf("migrating publisher config: %w", err)
	}
	executorConfig, err := migrateEngines(in.Node)
	if err != nil {
		return types.Bacalhau{}, fmt.Errorf("migrating executor config: %w", err)
	}
	inputSourceConfig, err := migrateInputSources(in.Node)
	if err != nil {
		return types.Bacalhau{}, fmt.Errorf("migrating input source config: %w", err)
	}
	downloaderConfig, err := migrateDownloadConfig(in.Node)
	if err != nil {
		return types.Bacalhau{}, fmt.Errorf("migrating result downloader config: %w", err)
	}
	out := types.Bacalhau{
		DataDir: "~/.bacalhau",
		// TODO(forrest) [review]: when migrating should the address come from the server or client when both are present?
		API: migrateAPI(in),
		Orchestrator: types.Orchestrator{
			Enabled: slices.ContainsFunc(in.Node.Type, func(s string) bool {
				return strings.ToLower(s) == "requester"
			}),
			Advertise: in.Node.Network.AdvertisedAddress,
			Cluster:   migrateCluster(in.Node.Network.Cluster),
			NodeManager: types.NodeManager{
				DisconnectTimeout: types.Duration(in.Node.Requester.ControlPlaneSettings.NodeDisconnectedAfter),
				ManualApproval:    in.Node.Requester.ManualNodeApproval,
			},
			Scheduler: types.Scheduler{
				WorkerCount:          in.Node.Requester.Worker.WorkerCount,
				HousekeepingInterval: types.Duration(in.Node.Requester.HousekeepingBackgroundTaskInterval),
			},
			EvaluationBroker: types.EvaluationBroker{
				VisibilityTimeout: types.Duration(in.Node.Requester.EvaluationBroker.EvalBrokerVisibilityTimeout),
				MaxRetryCount:     in.Node.Requester.EvaluationBroker.EvalBrokerMaxRetryCount,
			},
		},
		Compute: types.Compute{
			Enabled: slices.ContainsFunc(in.Node.Type, func(s string) bool {
				return strings.ToLower(s) == "compute"
			}),
			Orchestrators: in.Node.Network.Orchestrators,
			Labels:        in.Node.Labels,
			Heartbeat: types.Heartbeat{
				Interval:               types.Duration(in.Node.Compute.ControlPlaneSettings.HeartbeatFrequency),
				ResourceUpdateInterval: types.Duration(in.Node.Compute.ControlPlaneSettings.ResourceUpdateFrequency),
				InfoUpdateInterval:     types.Duration(in.Node.Compute.ControlPlaneSettings.InfoUpdateFrequency),
			},
			Volumes: func(paths []string) []types.Volume {
				out := make([]types.Volume, len(paths))
				for i, p := range paths {
					out[i] = types.Volume{
						Name: p,
						Path: p,
					}
				}
				return out
			}(in.Node.AllowListedLocalPaths),
		},
		WebUI: types.WebUI{
			Enabled: in.Node.WebUI.Enabled,
			Listen: func(enabled bool, port int) string {
				if enabled {
					return fmt.Sprintf("0.0.0.0:%d", port)
				}
				return ""
			}(in.Node.WebUI.Enabled, in.Node.WebUI.Port),
		},
		InputSources:      inputSourceConfig,
		Publishers:        publisherConfig,
		Executors:         executorConfig,
		ResultDownloaders: downloaderConfig,
		// TODO(forrest) [review]: currently both the compute and requester have a job selection policy
		// it is not clear whose policy should be migrated here.
		JobAdmissionControl: types.JobAdmissionControl{
			RejectStatelessJobs: in.Node.Requester.JobSelectionPolicy.RejectStatelessJobs,
			AcceptNetworkedJobs: in.Node.Requester.JobSelectionPolicy.AcceptNetworkedJobs,
			ProbeHTTP:           in.Node.Requester.JobSelectionPolicy.ProbeHTTP,
			ProbeExec:           in.Node.Requester.JobSelectionPolicy.ProbeExec,
		},
		Logging: types.Logging{
			Mode:                 string(in.Node.LoggingMode),
			LogDebugInfoInterval: types.Duration(in.Node.Compute.Logging.LogRunningExecutionsInterval),
		},
		UpdateConfig: types.UpdateConfig{
			Interval: types.Duration(in.Update.CheckFrequency),
		},
		FeatureFlags: types.FeatureFlags{
			ExecTranslation: in.Node.Requester.TranslationEnabled,
		},
	}
	return out, nil
}

func migrateCluster(in v1types.NetworkClusterConfig) types.Cluster {
	var out types.Cluster
	if in.Port != 0 {
		out.Listen = fmt.Sprintf("0.0.0.0:%d", in.Port)
	}
	if len(in.Peers) != 0 {
		out.Peers = in.Peers
	}
	if in.AdvertisedAddress != "" {
		in.AdvertisedAddress = out.Advertise
	}
	return out
}

// TODO(review): what api should we migrate, the server or the client?
func migrateAPI(in v1types.BacalhauConfig) types.API {
	protocol := "http"
	var (
		address string
		host    string
		port    int
	)
	// check for a client config
	if in.Node.ClientAPI.Host != "" && in.Node.ClientAPI.Port != 0 {
		if in.Node.ClientAPI.ClientTLS.UseTLS {
			protocol = "https"
		}
		host = in.Node.ClientAPI.Host
		port = in.Node.ClientAPI.Port

	}
	// check for server config
	if in.Node.ServerAPI.Host != "" && in.Node.ServerAPI.Port != 0 {
		if in.Node.ServerAPI.TLS.ServerKey != "" ||
			in.Node.ServerAPI.TLS.SelfSigned ||
			in.Node.ServerAPI.TLS.ServerCertificate != "" ||
			in.Node.ServerAPI.TLS.AutoCertCachePath != "" ||
			in.Node.ServerAPI.TLS.AutoCert != "" {
			protocol = "https"
		}
		host = in.Node.ServerAPI.Host
		port = in.Node.ServerAPI.Port
	}
	if host != "" && port != 0 {
		address = fmt.Sprintf("%s://%s:%d", protocol, host, port)
	}
	return types.API{
		Address: address,
		Auth: types.AuthConfig{
			TokensPath: in.Auth.TokensPath,
			Methods: func(cfg map[string]v1types.AuthenticatorConfig) map[string]types.AuthenticatorConfig {
				out := make(map[string]types.AuthenticatorConfig)
				for k, v := range cfg {
					out[k] = types.AuthenticatorConfig{
						Type:       string(v.Type),
						PolicyPath: v.PolicyPath,
					}
				}
				return out
			}(in.Auth.Methods),
			AccessPolicyPath: in.Auth.AccessPolicyPath,
		},
	}
}

func migrateDownloadConfig(in v1types.NodeConfig) (types.ResultDownloaders, error) {
	var out types.ResultDownloaders
	if in.DownloadURLRequestTimeout != 0 {
		out.Timeout = types.Duration(in.DownloadURLRequestTimeout)
	}
	if in.IPFS.Connect != "" {
		ipfsDownloaderCfg := types.IpfsDownloadConfig{Connect: in.IPFS.Connect}

		config := make(map[string]interface{})
		var dwnldCfg map[string]interface{}
		if err := mapstructure.Decode(ipfsDownloaderCfg, &dwnldCfg); err != nil {
			return types.ResultDownloaders{}, err
		}
		out.Config = make(map[string]map[string]interface{})
		out.Config[types.KindDownloadIPFS] = config
	}
	return out, nil
}

func migrateEngines(in v1types.NodeConfig) (types.ExecutorsConfig, error) {
	var out types.ExecutorsConfig
	// migrate any disabled engines
	out.Disabled = in.DisabledFeatures.Engines

	// only write a config if there exists at least one non-zero configuration value, otherwise omit this
	if in.Compute.ManifestCache.Size != 0 ||
		in.Compute.ManifestCache.Duration != 0 ||
		in.Compute.ManifestCache.Frequency != 0 {
		dockerConfig := types.Docker{
			ManifestCache: types.DockerManifestCache{
				Size:    in.Compute.ManifestCache.Size,
				TTL:     types.Duration(in.Compute.ManifestCache.Duration),
				Refresh: types.Duration(in.Compute.ManifestCache.Frequency),
			},
		}
		config := make(map[string]interface{})
		var cacheConfig map[string]interface{}
		if err := mapstructure.Decode(dockerConfig, &cacheConfig); err != nil {
			return types.ExecutorsConfig{}, err
		}
		out.Config = make(map[string]map[string]interface{})
		out.Config[types.KindExecutorDocker] = config
	}
	return out, nil
}

func migratePublishers(in v1types.NodeConfig) (types.PublishersConfig, error) {
	var out types.PublishersConfig
	// migrate any disabled publishers
	out.Disabled = in.DisabledFeatures.Publishers

	if in.IPFS.Connect != "" {
		ipfsPublisherCfg := types.IpfsPublisherConfig{Connect: in.IPFS.Connect}
		config := make(map[string]interface{})
		var ipfscfg map[string]interface{}
		if err := mapstructure.Decode(ipfsPublisherCfg, ipfscfg); err != nil {
			return types.PublishersConfig{}, err
		}
		out.Config = make(map[string]map[string]interface{})
		out.Config[types.KindPublisherIPFS] = config
	}

	if in.Compute.LocalPublisher.Address != "" ||
		in.Compute.LocalPublisher.Port != 0 ||
		in.Compute.LocalPublisher.Directory != "" {
		localPublisherCfg := types.LocalPublisherConfig{}
		if in.Compute.LocalPublisher.Address != "" {
			localPublisherCfg.Address = in.Compute.LocalPublisher.Address
		}
		if in.Compute.LocalPublisher.Port != 0 {
			localPublisherCfg.Port = in.Compute.LocalPublisher.Port
		}
		if in.Compute.LocalPublisher.Directory != "" {
			localPublisherCfg.Directory = in.Compute.LocalPublisher.Directory
		}
		var localcfg map[string]interface{}
		if err := mapstructure.Decode(localPublisherCfg, &localcfg); err != nil {
			return types.PublishersConfig{}, err
		}
		out.Config = make(map[string]map[string]interface{})
		out.Config[types.KindPublisherLocal] = localcfg
	}

	if in.Requester.StorageProvider.S3.PreSignedURLDisabled ||
		in.Requester.StorageProvider.S3.PreSignedURLExpiration != 0 {
		s3PublisherCfg := types.S3PublisherConfig{}
		if in.Requester.StorageProvider.S3.PreSignedURLDisabled {
			s3PublisherCfg.PreSignedURLDisabled = in.Requester.StorageProvider.S3.PreSignedURLDisabled
		}
		if in.Requester.StorageProvider.S3.PreSignedURLExpiration != 0 {
			s3PublisherCfg.PreSignedURLExpiration = types.Duration(in.Requester.StorageProvider.S3.PreSignedURLExpiration)
		}
		var s3cfg map[string]interface{}
		if err := mapstructure.Decode(s3PublisherCfg, &s3cfg); err != nil {
			return types.PublishersConfig{}, err
		}
		out.Config = make(map[string]map[string]interface{})
		out.Config[types.KindPublisherS3] = s3cfg
	}
	return out, nil
}

func migrateInputSources(in v1types.NodeConfig) (types.InputSourcesConfig, error) {
	var out types.InputSourcesConfig
	out.Disabled = in.DisabledFeatures.Storages
	if in.IPFS.Connect != "" {
		ipfsStorageCfg := types.IpfsInputSourceConfig{Connect: in.IPFS.Connect}
		config := make(map[string]interface{})
		var ipfscfg map[string]interface{}
		if err := mapstructure.Decode(ipfsStorageCfg, ipfscfg); err != nil {
			return types.InputSourcesConfig{}, err
		}
		out.Config = make(map[string]map[string]interface{})
		out.Config[types.KindStorageIPFS] = config
	}

	return out, nil
}
