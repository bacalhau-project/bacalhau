package configv2

import (
	"fmt"
	"slices"
	"strings"

	v1types "github.com/bacalhau-project/bacalhau/pkg/config/types"
	"github.com/bacalhau-project/bacalhau/pkg/configv2/types"
)

func MigrateV1(in v1types.BacalhauConfig) (types.Bacalhau, error) {
	out := types.Bacalhau{
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
		InputSources:      migrateInputSources(in.Node),
		Publishers:        migratePublishers(in.Node),
		Engines:           migrateEngines(in.Node),
		ResultDownloaders: migrateDownloadConfig(in.Node),
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
	var (
		host string
		port int
	)
	// check for a client config
	if in.Node.ClientAPI.Host != "" && in.Node.ClientAPI.Port != 0 {
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
		}
		host = in.Node.ServerAPI.Host
		port = in.Node.ServerAPI.Port
	}

	return types.API{
		Host: host,
		Port: port,
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

func migrateDownloadConfig(in v1types.NodeConfig) types.ResultDownloaders {
	var out types.ResultDownloaders

	out.Timeout = types.Duration(in.DownloadURLRequestTimeout)
	out.IPFS.Endpoint = in.IPFS.Connect

	return out
}

func migrateEngines(in v1types.NodeConfig) types.EngineConfig {
	var out types.EngineConfig

	// migrate any disabled engines
	out.Disabled = in.DisabledFeatures.Engines

	out.Docker.ManifestCache = types.DockerManifestCache{
		Size:    in.Compute.ManifestCache.Size,
		TTL:     types.Duration(in.Compute.ManifestCache.Duration),
		Refresh: types.Duration(in.Compute.ManifestCache.Frequency),
	}

	return out
}

func migratePublishers(in v1types.NodeConfig) types.PublishersConfig {
	var out types.PublishersConfig

	// migrate any disabled publishers
	out.Disabled = in.DisabledFeatures.Publishers

	// ipfs
	out.IPFS.Endpoint = in.IPFS.Connect

	// local
	out.Local.Port = in.Compute.LocalPublisher.Port
	out.Local.Address = in.Compute.LocalPublisher.Address
	out.Local.Directory = in.Compute.LocalPublisher.Directory

	// s3
	out.S3.PreSignedURLExpiration = types.Duration(in.Requester.StorageProvider.S3.PreSignedURLExpiration)
	out.S3.PreSignedURLDisabled = in.Requester.StorageProvider.S3.PreSignedURLDisabled

	return out
}

func migrateInputSources(in v1types.NodeConfig) types.InputSourcesConfig {
	var out types.InputSourcesConfig

	// migrate any disabled storages
	out.Disabled = in.DisabledFeatures.Storages

	// ipfs
	out.IPFS.Endpoint = in.IPFS.Connect

	return out
}
