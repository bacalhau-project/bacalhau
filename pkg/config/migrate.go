package config

import (
	"fmt"
	"slices"
	"strings"

	"github.com/bacalhau-project/bacalhau/pkg/config/cfgtypes"
	v1types "github.com/bacalhau-project/bacalhau/pkg/config_legacy/types"
)

func MigrateV1(in v1types.BacalhauConfig) (cfgtypes.Bacalhau, error) {
	out := cfgtypes.Bacalhau{
		// TODO(forrest) [review]: when migrating should the address come from the server or client when both are present?
		API: migrateAPI(in),
		Orchestrator: cfgtypes.Orchestrator{
			Enabled: slices.ContainsFunc(in.Node.Type, func(s string) bool {
				return strings.ToLower(s) == "requester"
			}),
			Advertise: in.Node.Network.AdvertisedAddress,
			Cluster:   migrateCluster(in.Node.Network.Cluster),
			NodeManager: cfgtypes.NodeManager{
				DisconnectTimeout: cfgtypes.Duration(in.Node.Requester.ControlPlaneSettings.NodeDisconnectedAfter),
				ManualApproval:    in.Node.Requester.ManualNodeApproval,
			},
			Scheduler: cfgtypes.Scheduler{
				WorkerCount:          in.Node.Requester.Worker.WorkerCount,
				HousekeepingInterval: cfgtypes.Duration(in.Node.Requester.HousekeepingBackgroundTaskInterval),
			},
			EvaluationBroker: cfgtypes.EvaluationBroker{
				VisibilityTimeout: cfgtypes.Duration(in.Node.Requester.EvaluationBroker.EvalBrokerVisibilityTimeout),
				MaxRetryCount:     in.Node.Requester.EvaluationBroker.EvalBrokerMaxRetryCount,
			},
		},
		Compute: cfgtypes.Compute{
			Enabled: slices.ContainsFunc(in.Node.Type, func(s string) bool {
				return strings.ToLower(s) == "compute"
			}),
			Orchestrators: in.Node.Network.Orchestrators,
			Labels:        in.Node.Labels,
			Heartbeat: cfgtypes.Heartbeat{
				Interval:               cfgtypes.Duration(in.Node.Compute.ControlPlaneSettings.HeartbeatFrequency),
				ResourceUpdateInterval: cfgtypes.Duration(in.Node.Compute.ControlPlaneSettings.ResourceUpdateFrequency),
				InfoUpdateInterval:     cfgtypes.Duration(in.Node.Compute.ControlPlaneSettings.InfoUpdateFrequency),
			},
			AllowListedLocalPaths: in.Node.AllowListedLocalPaths,
		},
		WebUI: cfgtypes.WebUI{
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
		JobAdmissionControl: func(config v1types.NodeConfig) cfgtypes.JobAdmissionControl {
			out := cfgtypes.JobAdmissionControl{
				RejectStatelessJobs: in.Node.Requester.JobSelectionPolicy.RejectStatelessJobs,
				AcceptNetworkedJobs: in.Node.Requester.JobSelectionPolicy.AcceptNetworkedJobs,
				ProbeHTTP:           in.Node.Requester.JobSelectionPolicy.ProbeHTTP,
				ProbeExec:           in.Node.Requester.JobSelectionPolicy.ProbeExec,
			}

			// compute node configuration takes precedence.
			if in.Node.Compute.JobSelection.RejectStatelessJobs {
				out.RejectStatelessJobs = in.Node.Compute.JobSelection.RejectStatelessJobs
			}
			if in.Node.Compute.JobSelection.AcceptNetworkedJobs {
				out.RejectStatelessJobs = in.Node.Compute.JobSelection.AcceptNetworkedJobs
			}
			if in.Node.Compute.JobSelection.ProbeHTTP != "" {
				out.ProbeHTTP = in.Node.Compute.JobSelection.ProbeHTTP
			}
			if in.Node.Compute.JobSelection.ProbeExec != "" {
				out.ProbeExec = in.Node.Compute.JobSelection.ProbeExec
			}

			return out
		}(in.Node),
		Logging: cfgtypes.Logging{
			Mode:                 string(in.Node.LoggingMode),
			LogDebugInfoInterval: cfgtypes.Duration(in.Node.Compute.Logging.LogRunningExecutionsInterval),
		},
		UpdateConfig: cfgtypes.UpdateConfig{
			Interval: cfgtypes.Duration(in.Update.CheckFrequency),
		},
		FeatureFlags: cfgtypes.FeatureFlags{
			ExecTranslation: in.Node.Requester.TranslationEnabled,
		},
	}
	return out, nil
}

func migrateCluster(in v1types.NetworkClusterConfig) cfgtypes.Cluster {
	var out cfgtypes.Cluster
	if in.Port != 0 {
		out.Port = in.Port
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
func migrateAPI(in v1types.BacalhauConfig) cfgtypes.API {
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

	return cfgtypes.API{
		Host: host,
		Port: port,
		Auth: cfgtypes.AuthConfig{
			TokensPath: in.Auth.TokensPath,
			Methods: func(cfg map[string]v1types.AuthenticatorConfig) map[string]cfgtypes.AuthenticatorConfig {
				out := make(map[string]cfgtypes.AuthenticatorConfig)
				for k, v := range cfg {
					out[k] = cfgtypes.AuthenticatorConfig{
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

func migrateDownloadConfig(in v1types.NodeConfig) cfgtypes.ResultDownloaders {
	var out cfgtypes.ResultDownloaders

	out.Timeout = cfgtypes.Duration(in.DownloadURLRequestTimeout)
	out.IPFS.Endpoint = in.IPFS.Connect

	return out
}

func migrateEngines(in v1types.NodeConfig) cfgtypes.EngineConfig {
	var out cfgtypes.EngineConfig

	// migrate any disabled engines
	out.Disabled = in.DisabledFeatures.Engines

	out.Docker.ManifestCache = cfgtypes.DockerManifestCache{
		Size:    in.Compute.ManifestCache.Size,
		TTL:     cfgtypes.Duration(in.Compute.ManifestCache.Duration),
		Refresh: cfgtypes.Duration(in.Compute.ManifestCache.Frequency),
	}

	return out
}

func migratePublishers(in v1types.NodeConfig) cfgtypes.PublishersConfig {
	var out cfgtypes.PublishersConfig

	// migrate any disabled publishers
	out.Disabled = in.DisabledFeatures.Publishers

	// ipfs
	out.IPFS.Endpoint = in.IPFS.Connect

	// local
	out.Local.Port = in.Compute.LocalPublisher.Port
	out.Local.Address = in.Compute.LocalPublisher.Address
	out.Local.Directory = in.Compute.LocalPublisher.Directory

	// s3
	out.S3.PreSignedURLExpiration = cfgtypes.Duration(in.Requester.StorageProvider.S3.PreSignedURLExpiration)
	out.S3.PreSignedURLDisabled = in.Requester.StorageProvider.S3.PreSignedURLDisabled

	return out
}

func migrateInputSources(in v1types.NodeConfig) cfgtypes.InputSourcesConfig {
	var out cfgtypes.InputSourcesConfig

	// migrate any disabled storages
	out.Disabled = in.DisabledFeatures.Storages

	// ipfs
	out.IPFS.Endpoint = in.IPFS.Connect

	return out
}
